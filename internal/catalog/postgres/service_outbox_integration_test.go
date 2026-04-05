package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type failingOutbox struct {
	err error
}

func (o failingOutbox) Add(ctx context.Context, events []catalog.Event) error {
	return o.err
}

func newPublishedLotForServiceTest(t *testing.T, lotID, auctionID string) *catalog.Lot {
	t.Helper()

	lot, _, err := catalog.NewLot(
		lotID,
		"prod-1",
		"seller-1",
		"photo-key",
		10,
		100,
		catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour), time.Hour),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := lot.AssignAuctionID(auctionID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := lot.Publish(true, catalog.ProductSnapshot{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return lot
}

func TestCloseLotRollsBackWhenOutboxFails(t *testing.T) {
	db := openIntegrationDB(t, "service-rollback")
	lotRepo := NewLotRepository(db)
	outboxErr := errors.New("outbox failed")
	service := app.NewCatalogService(
		nil,
		nil,
		nil,
		nil,
		lotRepo,
		failingOutbox{err: outboxErr},
		nil,
		NewTransactionManager(db, nil),
	)

	lot := newPublishedLotForServiceTest(t, "lot-rb", "auc-rb")
	if err := lotRepo.Save(context.Background(), lot); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	err := service.CloseLot(context.Background(), "lot-rb", 150)
	if !errors.Is(err, outboxErr) {
		t.Fatalf("expected outbox error, got %v", err)
	}

	stored, err := lotRepo.Get(context.Background(), "lot-rb")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if stored.Status() != catalog.LotStatusPublished {
		t.Fatalf("expected rollback to preserve published status, got %s", stored.Status())
	}

	store := testIntegrationDriver.store("service-rollback")
	if len(store.outbox) != 0 {
		t.Fatalf("expected no committed outbox messages, got %d", len(store.outbox))
	}
}

func TestCloseLotCommitsAggregateAndOutbox(t *testing.T) {
	db := openIntegrationDB(t, "service-commit")
	lotRepo := NewLotRepository(db)
	outboxRepo := NewOutboxRepository(db)
	service := app.NewCatalogService(
		nil,
		nil,
		nil,
		nil,
		lotRepo,
		outboxRepo,
		nil,
		NewTransactionManager(db, nil),
	)

	lot := newPublishedLotForServiceTest(t, "lot-ok", "auc-ok")
	if err := lotRepo.Save(context.Background(), lot); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if err := service.CloseLot(context.Background(), "lot-ok", 175); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	stored, err := lotRepo.Get(context.Background(), "lot-ok")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if stored.Status() != catalog.LotStatusClosed {
		t.Fatalf("expected closed status, got %s", stored.Status())
	}
	if stored.FinalPrice() != 175 {
		t.Fatalf("expected final price 175, got %d", stored.FinalPrice())
	}

	store := testIntegrationDriver.store("service-commit")
	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 committed outbox message, got %d", len(store.outbox))
	}
	if store.outbox[0].eventType != "catalog.LotClosed" {
		t.Fatalf("expected catalog.LotClosed event type, got %s", store.outbox[0].eventType)
	}
	if store.outbox[0].aggregateID != "lot-ok" {
		t.Fatalf("expected aggregate id lot-ok, got %s", store.outbox[0].aggregateID)
	}
}
