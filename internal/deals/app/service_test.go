package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"unbelievable_fish/internal/deals/deal"
)

type dealRepoStub struct {
	byID      map[string]*deal.Deal
	byAuction map[string]*deal.Deal
	saved     []*deal.Deal
}

func newDealRepoStub() *dealRepoStub {
	return &dealRepoStub{
		byID:      make(map[string]*deal.Deal),
		byAuction: make(map[string]*deal.Deal),
	}
}

func (r *dealRepoStub) Save(_ context.Context, item *deal.Deal) error {
	r.byID[item.ID()] = item
	if item.AuctionID() != "" {
		r.byAuction[item.AuctionID()] = item
	}
	r.saved = append(r.saved, item)
	return nil
}

func (r *dealRepoStub) GetByID(_ context.Context, dealID string) (*deal.Deal, error) {
	item, ok := r.byID[dealID]
	if !ok {
		return nil, ErrDealNotFound
	}
	return item, nil
}

func (r *dealRepoStub) GetByAuctionID(_ context.Context, auctionID string) (*deal.Deal, error) {
	item, ok := r.byAuction[auctionID]
	if !ok {
		return nil, ErrDealNotFound
	}
	return item, nil
}

type projectionRepoStub struct {
	byAuction map[string]*deal.DealProjection
	saved     []*deal.DealProjection
}

func newProjectionRepoStub() *projectionRepoStub {
	return &projectionRepoStub{byAuction: make(map[string]*deal.DealProjection)}
}

func (r *projectionRepoStub) Save(_ context.Context, item *deal.DealProjection) error {
	r.byAuction[item.AuctionID] = item
	r.saved = append(r.saved, item)
	return nil
}

func (r *projectionRepoStub) GetByAuctionID(_ context.Context, auctionID string) (*deal.DealProjection, error) {
	item, ok := r.byAuction[auctionID]
	if !ok {
		return nil, deal.ErrProjectionNotFound
	}
	return item, nil
}

type publisherStub struct {
	published [][]deal.Event
}

func (p *publisherStub) Publish(_ context.Context, events []deal.Event) error {
	copied := make([]deal.Event, len(events))
	copy(copied, events)
	p.published = append(p.published, copied)
	return nil
}

func TestService_CreateProjectionFromLotPublished(t *testing.T) {
	logTest(t)

	svc := newTestService(t)
	now := time.Now()
	logMsg(t, "creating projection for auction=auc_1 supplier=sup_1")

	projection, err := svc.CreateProjectionFromLotPublished(context.Background(), CreateProjectionCommand{
		AuctionID:  "auc_1",
		SupplierID: "sup_1",
		ProductSnapshot: deal.ProductSnapshot{
			ProductID: "prod_1",
			Name:      "Fish",
		},
		StartPrice:  100,
		PublishedAt: now,
	})
	if err != nil {
		logMsg(t, "create projection error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "projection created status="+string(projection.Status))

	if projection.AuctionID != "auc_1" {
		t.Fatalf("expected projection for auc_1, got %s", projection.AuctionID)
	}
	if len(svc.projections.saved) != 1 {
		t.Fatalf("expected 1 saved projection, got %d", len(svc.projections.saved))
	}
}

func TestService_CreateDealFromAuctionWon(t *testing.T) {
	logTest(t)

	svc := newTestService(t)
	now := time.Now()
	projection := deal.NewDealProjection(
		"auc_1",
		"sup_1",
		deal.ProductSnapshot{ProductID: "prod_1", Name: "Fish"},
		100,
		now.Add(-time.Hour),
	)
	svc.projections.byAuction[projection.AuctionID] = projection
	logMsg(t, "projection prepared for auction="+projection.AuctionID)

	item, err := svc.CreateDealFromAuctionWon(context.Background(), CreateDealFromAuctionWonCommand{
		AuctionID:       "auc_1",
		WinnerCompanyID: "cust_1",
		FinalPrice:      150,
		WonAt:           now,
	})
	if err != nil {
		logMsg(t, "create deal error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "deal created id="+item.ID()+" status="+string(item.Status()))

	if item.Status() != deal.DealStatusPending {
		t.Fatalf("expected pending status, got %s", item.Status())
	}
	if projection.Status != deal.ProjectionStatusConverted {
		t.Fatalf("expected converted projection, got %s", projection.Status)
	}
	if len(svc.deals.saved) != 1 {
		t.Fatalf("expected 1 saved deal, got %d", len(svc.deals.saved))
	}
	if len(svc.publisher.published) != 1 {
		t.Fatalf("expected 1 published batch, got %d", len(svc.publisher.published))
	}
}

func TestService_ConfirmDeal(t *testing.T) {
	logTest(t)

	svc := newTestService(t)
	item := createPendingDeal(t)
	svc.deals.byID[item.ID()] = item
	logMsg(t, "pending deal id="+item.ID())

	updated, err := svc.ConfirmDeal(context.Background(), ConfirmDealCommand{DealID: item.ID()})
	if err != nil {
		logMsg(t, "confirm error="+err.Error())
		t.Fatalf("unexpected error: %v", err)
	}
	logMsg(t, "deal confirmed status="+string(updated.Status()))

	if updated.Status() != deal.DealStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", updated.Status())
	}
	if len(svc.publisher.published) != 1 {
		t.Fatalf("expected 1 published batch, got %d", len(svc.publisher.published))
	}
}

func TestService_RequestPayment_RequiresInvoiceNumber(t *testing.T) {
	logTest(t)

	svc := newTestService(t)
	logMsg(t, "request payment without invoice for deal=deal_1")

	_, err := svc.RequestPayment(context.Background(), RequestPaymentCommand{DealID: "deal_1"})
	if !errors.Is(err, ErrInvoiceNumberRequired) {
		logMsg(t, "unexpected error="+err.Error())
		t.Fatalf("expected ErrInvoiceNumberRequired, got %v", err)
	}
	logMsg(t, "validation error matched ErrInvoiceNumberRequired")
}

func newTestService(t *testing.T) *testService {
	t.Helper()

	deals := newDealRepoStub()
	projections := newProjectionRepoStub()
	publisher := &publisherStub{}

	svc, err := NewService(deals, projections, publisher)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	return &testService{
		Service:     svc,
		deals:       deals,
		projections: projections,
		publisher:   publisher,
	}
}

type testService struct {
	*Service
	deals       *dealRepoStub
	projections *projectionRepoStub
	publisher   *publisherStub
}

func createPendingDeal(t *testing.T) *deal.Deal {
	t.Helper()

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc_test",
		"sup_test",
		deal.ProductSnapshot{ProductID: "prod_test", Name: "Fish"},
		100,
		time.Now().Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "cust_test", 120, time.Now())
	if err != nil {
		t.Fatalf("unexpected factory error: %v", err)
	}
	return item
}
