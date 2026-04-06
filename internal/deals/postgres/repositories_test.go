package postgres

import (
	"context"
	"testing"
	"time"

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
