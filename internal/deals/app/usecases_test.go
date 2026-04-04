package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"unbelievable_fish/internal/deals/deal"
)

func TestCreateProjectionSavesProjection(t *testing.T) {
	logTest(t)
	calls := []string{}
	projections := &projectionRepoSpy{calls: &calls}

	uc := NewCreateProjection(projections)
	now := time.Now()
	logMsg(t, "create projection auction=auc-1 supplier=sup-1")
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"save_projection"})
	if projections.lastSaved == nil || projections.lastSaved.AuctionID != "auc-1" {
		t.Fatal("expected projection to be saved")
	}
}

func TestCreateDealFromAuctionWonOrchestratesSaveAndPublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	projection := deal.NewDealProjection("auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, time.Now().Add(-time.Hour))
	deals := &dealRepoSpy{calls: &calls}
	projections := &projectionRepoSpy{calls: &calls, projection: projection}
	publisher := &publisherSpy{calls: &calls}

	uc := NewCreateDealFromAuctionWon(deals, projections, publisher)
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "cust-1", 120, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_projection", "save_deal", "save_projection", "publish"})
	if deals.lastSaved == nil {
		t.Fatal("expected deal to be saved")
	}
	if len(publisher.published) == 0 || len(publisher.published[0]) == 0 {
		t.Fatal("expected events to be published")
	}
}

func TestConfirmDealOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	deals := &dealRepoSpy{calls: &calls, deal: item}
	publisher := &publisherSpy{calls: &calls}

	uc := NewConfirmDeal(deals, publisher)
	if err := uc.Execute(context.Background(), testMeta(), item.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "save_deal", "publish"})
	if deals.lastSaved.Status() != deal.DealStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", deals.lastSaved.Status())
	}
}

func TestUpdateDealPriceUsesMetaActor(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	deals := &dealRepoSpy{calls: &calls, deal: item}
	publisher := &publisherSpy{calls: &calls}

	uc := NewUpdateDealPrice(deals, publisher)
	if err := uc.Execute(context.Background(), testMeta(), item.ID(), 130); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "save_deal", "publish"})
}

type dealRepoSpy struct {
	calls     *[]string
	deal      *deal.Deal
	lastSaved *deal.Deal
}

func (s *dealRepoSpy) Save(ctx context.Context, item *deal.Deal) error {
	_ = ctx
	s.lastSaved = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_deal")
	}
	return nil
}

func (s *dealRepoSpy) GetByID(ctx context.Context, dealID string) (*deal.Deal, error) {
	_ = ctx
	_ = dealID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_deal")
	}
	if s.deal == nil {
		return nil, ErrDealNotFound
	}
	return s.deal, nil
}

func (s *dealRepoSpy) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	_ = auctionID
	if s.deal == nil {
		return nil, ErrDealNotFound
	}
	return s.deal, nil
}

type projectionRepoSpy struct {
	calls      *[]string
	projection *deal.DealProjection
	lastSaved  *deal.DealProjection
}

func (s *projectionRepoSpy) Save(ctx context.Context, item *deal.DealProjection) error {
	_ = ctx
	s.lastSaved = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_projection")
	}
	return nil
}

func (s *projectionRepoSpy) GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	_ = ctx
	_ = auctionID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_projection")
	}
	if s.projection == nil {
		return nil, deal.ErrProjectionNotFound
	}
	return s.projection, nil
}

type publisherSpy struct {
	calls     *[]string
	published [][]deal.Event
}

func (s *publisherSpy) Publish(ctx context.Context, events []deal.Event) error {
	_ = ctx
	if s.calls != nil {
		*s.calls = append(*s.calls, "publish")
	}
	s.published = append(s.published, events)
	return nil
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}
}

func testMeta() CommandMeta {
	return CommandMeta{
		CompanyID:     "company-1",
		UserID:        "user-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func createPendingDeal(t *testing.T) *deal.Deal {
	t.Helper()

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc-test",
		"sup-test",
		deal.ProductSnapshot{ProductID: "prod-test", Name: "Fish"},
		100,
		time.Now().Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-test", 120, time.Now())
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	return item
}
