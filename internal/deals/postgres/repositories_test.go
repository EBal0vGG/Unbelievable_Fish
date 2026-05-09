package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

func TestDealRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "deal-repo")
	repo := NewDealRepository(db)

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc-1",
		"sup-1",
		deal.ProductSnapshot{ProductID: "prod-1", Name: "Fish", Description: "Desc", Category: "cat"},
		100,
		time.Now().Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-1", 120, time.Now())
	if err != nil {
		t.Fatalf("create deal error: %v", err)
	}

	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.GetByID(context.Background(), item.ID())
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ID() != item.ID() || loaded.CustomerID() != "buyer-1" {
		t.Fatalf("unexpected deal loaded: %+v", loaded)
	}
}

func TestDealRepositoryGetActiveDealByAuctionIDSkipsCancelled(t *testing.T) {
	db := openIntegrationDB(t, "deal-repo-by-auction")
	repo := NewDealRepository(db)
	ctx := context.Background()
	createdAt := time.Now().UTC()

	cancelledDeal, err := deal.Rehydrate(deal.RehydrateParams{
		ID:         "deal-cancelled",
		CustomerID: "buyer-old",
		SupplierID: "sup-1",
		AuctionID:  "auc-shared",
		Quantity:   1,
		UnitPrice:  100,
		Status:     deal.DealStatusCancelled,
		TypeName:   deal.DealTypeAuction,
		CreatedAt:  createdAt,
		ProductSnapshot: deal.ProductSnapshot{
			ProductID:   "prod-1",
			Name:        "Fish",
			Description: "Desc",
			Category:    "cat",
		},
	})
	if err != nil {
		t.Fatalf("rehydrate cancelled deal error: %v", err)
	}
	if err := repo.Save(ctx, cancelledDeal); err != nil {
		t.Fatalf("save cancelled deal error: %v", err)
	}

	pendingDeal, err := deal.Rehydrate(deal.RehydrateParams{
		ID:         "deal-pending",
		CustomerID: "buyer-next",
		SupplierID: "sup-1",
		AuctionID:  "auc-shared",
		Quantity:   1,
		UnitPrice:  100,
		Status:     deal.DealStatusPending,
		TypeName:   deal.DealTypeAuction,
		CreatedAt:  createdAt,
		ProductSnapshot: deal.ProductSnapshot{
			ProductID:   "prod-1",
			Name:        "Fish",
			Description: "Desc",
			Category:    "cat",
		},
	})
	if err != nil {
		t.Fatalf("rehydrate pending deal error: %v", err)
	}
	if err := repo.Save(ctx, pendingDeal); err != nil {
		t.Fatalf("save pending deal error: %v", err)
	}

	loaded, err := repo.GetActiveDealByAuctionID(ctx, "auc-shared")
	if err != nil {
		t.Fatalf("get active by auction error: %v", err)
	}
	if loaded.ID() != "deal-pending" {
		t.Fatalf("expected pending deal to be selected, got %s", loaded.ID())
	}
	if loaded.Status() != deal.DealStatusPending {
		t.Fatalf("expected status pending, got %s", loaded.Status())
	}
}

func TestDealRepositoryGetActiveDealByAuctionIDMultipleActiveErrors(t *testing.T) {
	db := openIntegrationDB(t, "deal-repo-multi-active")
	repo := NewDealRepository(db)
	ctx := context.Background()
	createdAt := time.Now().UTC()
	base := deal.RehydrateParams{
		SupplierID: "sup-1",
		AuctionID:  "auc-multi",
		Quantity:   1,
		UnitPrice:  100,
		Status:     deal.DealStatusPending,
		TypeName:   deal.DealTypeAuction,
		CreatedAt:  createdAt,
		ProductSnapshot: deal.ProductSnapshot{
			ProductID: "prod-1", Name: "Fish", Description: "Desc", Category: "cat",
		},
	}
	for _, id := range []string{"deal-a", "deal-b"} {
		p := base
		p.ID = id
		p.CustomerID = id + "-buyer"
		item, err := deal.Rehydrate(p)
		if err != nil {
			t.Fatalf("rehydrate: %v", err)
		}
		if err := repo.Save(ctx, item); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	_, err := repo.GetActiveDealByAuctionID(ctx, "auc-multi")
	if !errors.Is(err, dealsapp.ErrMultipleActiveDealsForAuction) {
		t.Fatalf("expected ErrMultipleActiveDealsForAuction, got %v", err)
	}
}

func TestProjectionRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "projection-repo")
	repo := NewProjectionRepository(db)

	item := deal.NewDealProjection(
		"auc-2",
		"sup-2",
		deal.ProductSnapshot{ProductID: "prod-2", Name: "Fish", Description: "Desc", Category: "cat"},
		150,
		time.Now(),
	)
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.GetByAuctionID(context.Background(), "auc-2")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.AuctionID != "auc-2" || loaded.SupplierID != "sup-2" {
		t.Fatalf("unexpected projection loaded: %+v", loaded)
	}
}

func TestDealConfirmationRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "confirmation-repo")
	repo := NewDealConfirmationRepository(db)

	item, err := deal.NewDealConfirmation(deal.DealConfirmationParams{
		DealID:                "deal-1",
		Stage:                 deal.DealConfirmationStageConfirmed,
		RequestedByUserID:     "user-1",
		RequestedByCompanyID:  "seller-1",
		CounterpartyCompanyID: "buyer-1",
		VerificationMethod:    deal.VerificationMethodManual,
		RequestedAt:           time.Now().UTC(),
		Comment:               "ready",
	})
	if err != nil {
		t.Fatalf("new confirmation error: %v", err)
	}

	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.GetByID(context.Background(), item.ID())
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ID() != item.ID() || loaded.DealID() != "deal-1" || loaded.Stage() != deal.DealConfirmationStageConfirmed {
		t.Fatalf("unexpected confirmation loaded: %+v", loaded)
	}
}

func TestSelectionRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "selection-repo")
	repo := NewSelectionRepository(db)

	item := deal.NewWinnerSelection(
		"auc-3",
		[]string{"c1", "c2"},
		200,
		time.Now(),
		"sup-3",
		deal.ProductSnapshot{ProductID: "prod-3", Name: "Fish", Description: "Desc", Category: "cat"},
	)
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.GetByAuctionID(context.Background(), "auc-3")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.AuctionID != "auc-3" || len(loaded.Candidates) != 2 {
		t.Fatalf("unexpected selection loaded: %+v", loaded)
	}
}

func TestSelectionRepositoryGetByAuctionIDForUpdateInTx(t *testing.T) {
	db := openIntegrationDB(t, "selection-repo-for-update")
	repo := NewSelectionRepository(db)
	ctx := context.Background()

	item := deal.NewWinnerSelection(
		"auc-lock",
		[]string{"c1"},
		200,
		time.Now().UTC(),
		"sup-lock",
		deal.ProductSnapshot{ProductID: "prod-lock", Name: "Fish", Description: "D", Category: "cat"},
	)
	if err := repo.Save(ctx, item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	tm := NewTransactionManager(db, nil)
	if err := tm.WithinTx(ctx, func(txCtx context.Context) error {
		loaded, err := repo.GetByAuctionIDForUpdate(txCtx, "auc-lock")
		if err != nil {
			return err
		}
		if loaded.AuctionID != "auc-lock" {
			t.Fatalf("unexpected row: %+v", loaded)
		}
		return nil
	}); err != nil {
		t.Fatalf("within tx: %v", err)
	}
}

func TestDealRepositoryGetByIDForUpdateInTx(t *testing.T) {
	db := openIntegrationDB(t, "deal-repo-for-update")
	repo := NewDealRepository(db)
	ctx := context.Background()

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc-lock-d",
		"sup-1",
		deal.ProductSnapshot{ProductID: "p", Name: "Fish", Description: "D", Category: "cat"},
		100,
		time.Now().Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-1", 120, time.Now())
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}
	if err := repo.Save(ctx, item); err != nil {
		t.Fatalf("save: %v", err)
	}

	tm := NewTransactionManager(db, nil)
	if err := tm.WithinTx(ctx, func(txCtx context.Context) error {
		loaded, err := repo.GetByIDForUpdate(txCtx, item.ID())
		if err != nil {
			return err
		}
		if loaded.ID() != item.ID() {
			t.Fatalf("unexpected deal id %s", loaded.ID())
		}
		return nil
	}); err != nil {
		t.Fatalf("within tx: %v", err)
	}
}
