package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
)

func TestOutboxRepositoryWorksOutsideTransaction(t *testing.T) {
	db := openIntegrationDB(t, "outbox-outside")
	repo := NewOutboxRepository(db)

	err := repo.Add(context.Background(), []catalog.Event{
		catalog.LotClosed{LotID: "lot-1", FinalPrice: 150, Status: catalog.LotStatusClosed},
	})
	if err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}

	store := testIntegrationDriver.store("outbox-outside")
	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 outbox message, got %d", len(store.outbox))
	}
	if store.outbox[0].eventType != "catalog.LotClosed" {
		t.Fatalf("expected event type catalog.LotClosed, got %s", store.outbox[0].eventType)
	}
	if store.outbox[0].aggregateID != "lot-1" {
		t.Fatalf("expected aggregate id lot-1, got %s", store.outbox[0].aggregateID)
	}
}

func TestOutboxRepositoryUsesTransactionCommitAndRollback(t *testing.T) {
	db := openIntegrationDB(t, "outbox-tx")
	repo := NewOutboxRepository(db)
	txManager := NewTransactionManager(db, nil)

	err := txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		return repo.Add(ctx, []catalog.Event{
			catalog.LotClosed{LotID: "lot-2", FinalPrice: 170, Status: catalog.LotStatusClosed},
		})
	})
	if err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}

	store := testIntegrationDriver.store("outbox-tx")
	if len(store.outbox) != 1 {
		t.Fatalf("expected 1 committed outbox message, got %d", len(store.outbox))
	}

	rollbackErr := errors.New("rollback outbox")
	err = txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := repo.Add(ctx, []catalog.Event{
			catalog.LotClosed{LotID: "lot-3", FinalPrice: 190, Status: catalog.LotStatusClosed},
		}); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}

	if len(store.outbox) != 1 {
		t.Fatalf("expected rollback to keep outbox size at 1, got %d", len(store.outbox))
	}
}
