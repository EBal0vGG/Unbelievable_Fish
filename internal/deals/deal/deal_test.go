package deal

import (
	"testing"
)

// Helper function для создания тестового productSnapshot
func createTestProductSnapshot() productSnapshot {
	return productSnapshot{
		ProductID:     "prod_123",
		Name:          "Test Product",
		Description:   "Test Description",
		Category:      "Test Category",
		Weight:        1.0,
		Volume:        1.0,
		OriginCountry: "Test Country",
	}
}

func TestNewDealFromLotPublished(t *testing.T) {
	tests := []struct {
		name       string
		auctionID  string
		sellerID   string
		snapshot   productSnapshot
		startPrice int64
		wantErr    error
	}{
		{
			name:       "success",
			auctionID:  "auction_1",
			sellerID:   "seller_1",
			snapshot:   createTestProductSnapshot(),
			startPrice: 1000,
			wantErr:    nil,
		},
		{
			name:       "empty auction id",
			auctionID:  "",
			sellerID:   "seller_1",
			snapshot:   createTestProductSnapshot(),
			startPrice: 1000,
			wantErr:    ErrAuctionIDRequired,
		},
		{
			name:       "empty seller id",
			auctionID:  "auction_1",
			sellerID:   "",
			snapshot:   createTestProductSnapshot(),
			startPrice: 1000,
			wantErr:    ErrSellerCompanyRequired,
		},
		{
			name:      "empty product name",
			auctionID: "auction_1",
			sellerID:  "seller_1",
			snapshot: productSnapshot{
				ProductID: "prod_123",
				Name:      "",
			},
			startPrice: 1000,
			wantErr:    ErrProductNameRequired,
		},
		{
			name:       "negative price",
			auctionID:  "auction_1",
			sellerID:   "seller_1",
			snapshot:   createTestProductSnapshot(),
			startPrice: -100,
			wantErr:    ErrPriceMustBePositive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deal, err := NewDealFromLotPublished(tt.auctionID, tt.sellerID, tt.snapshot, tt.startPrice)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("want error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if deal == nil {
				t.Error("deal should not be nil")
			}

			if deal.AuctionID() != tt.auctionID {
				t.Errorf("want auction id %s, got %s", tt.auctionID, deal.AuctionID())
			}

			if deal.Status() != DealStatusDrafted {
				t.Errorf("want status %s, got %s", DealStatusDrafted, deal.Status())
			}
		})
	}
}

func TestCompleteDealFromAuctionWon(t *testing.T) {
	deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)

	t.Run("success", func(t *testing.T) {
		result, err := CompleteDealFromAuctionWon(deal, "winner_1", 1500)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result.CustomerID() != "winner_1" {
			t.Errorf("want customer id winner_1, got %s", result.CustomerID())
		}

		if result.UnitPrice() != 1500 {
			t.Errorf("want price 1500, got %d", result.UnitPrice())
		}

		if result.Status() != DealStatusPending {
			t.Errorf("want status %s, got %s", DealStatusPending, result.Status())
		}
	})

	t.Run("not a draft deal", func(t *testing.T) {
		deal2, _ := NewDealFromLotPublished("auction_2", "seller_2", createTestProductSnapshot(), 1000)
		deal2.status = DealStatusConfirmed

		_, err := CompleteDealFromAuctionWon(deal2, "winner_1", 1500)
		if err != ErrOnlyDraftCanBeCompleted {
			t.Errorf("want error %v, got %v", ErrOnlyDraftCanBeCompleted, err)
		}
	})

	t.Run("empty winner id", func(t *testing.T) {
		deal2, _ := NewDealFromLotPublished("auction_2", "seller_2", createTestProductSnapshot(), 1000)

		_, err := CompleteDealFromAuctionWon(deal2, "", 1500)
		if err != ErrWinnerCompanyRequired {
			t.Errorf("want error %v, got %v", ErrWinnerCompanyRequired, err)
		}
	})
}

func TestDealConfirm(t *testing.T) {
	t.Run("confirm draft deal", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)

		err := deal.Confirm()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusConfirmed {
			t.Errorf("want status %s, got %s", DealStatusConfirmed, deal.Status())
		}

		if deal.ConfirmedAt() == nil {
			t.Error("confirmed at should be set")
		}
	})

	t.Run("cannot confirm cancelled deal", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
		deal.status = DealStatusCancelled

		err := deal.Confirm()
		if err != ErrCannotConfirmDeal {
			t.Errorf("want error %v, got %v", ErrCannotConfirmDeal, err)
		}
	})
}

func TestDealPaymentFlow(t *testing.T) {
	deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
	CompleteDealFromAuctionWon(deal, "winner_1", 1500)
	deal.Confirm()

	t.Run("request payment", func(t *testing.T) {
		err := deal.RequestPayment()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusPaymentRequested {
			t.Errorf("want status %s, got %s", DealStatusPaymentRequested, deal.Status())
		}
	})

	t.Run("mark as paid", func(t *testing.T) {
		err := deal.MarkAsPaid()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusPaid {
			t.Errorf("want status %s, got %s", DealStatusPaid, deal.Status())
		}
	})

	t.Run("cannot mark as paid without payment request", func(t *testing.T) {
		deal2, _ := NewDealFromLotPublished("auction_2", "seller_2", createTestProductSnapshot(), 1000)
		CompleteDealFromAuctionWon(deal2, "winner_2", 1500)
		deal2.Confirm()

		err := deal2.MarkAsPaid()
		if err != ErrCannotMarkAsPaid {
			t.Errorf("want error %v, got %v", ErrCannotMarkAsPaid, err)
		}
	})
}

func TestDealShipmentFlow(t *testing.T) {
	deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
	CompleteDealFromAuctionWon(deal, "winner_1", 1500)
	deal.Confirm()
	deal.RequestPayment()
	deal.MarkAsPaid()

	t.Run("request shipment", func(t *testing.T) {
		err := deal.RequestShipment()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusShipmentRequested {
			t.Errorf("want status %s, got %s", DealStatusShipmentRequested, deal.Status())
		}
	})

	t.Run("mark as shipped", func(t *testing.T) {
		err := deal.MarkAsShipped()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusShipped {
			t.Errorf("want status %s, got %s", DealStatusShipped, deal.Status())
		}
	})

	t.Run("complete deal", func(t *testing.T) {
		err := deal.Complete()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusCompleted {
			t.Errorf("want status %s, got %s", DealStatusCompleted, deal.Status())
		}
	})
}

func TestDealCancel(t *testing.T) {
	t.Run("cancel draft deal", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)

		err := deal.Cancel()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if deal.Status() != DealStatusCancelled {
			t.Errorf("want status %s, got %s", DealStatusCancelled, deal.Status())
		}
	})

	t.Run("cannot cancel completed deal", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
		deal.status = DealStatusCompleted

		err := deal.Cancel()
		if err != ErrCannotCancelDeal {
			t.Errorf("want error %v, got %v", ErrCannotCancelDeal, err)
		}
	})
}

func TestDealCalculateTotal(t *testing.T) {
	deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
	CompleteDealFromAuctionWon(deal, "winner_1", 1500)

	total := deal.CalculateTotal()
	if total != 1500 {
		t.Errorf("want total 1500, got %d", total)
	}
}

func TestDealValidate(t *testing.T) {
	t.Run("valid deal", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)

		err := deal.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid without customer id for non-draft", func(t *testing.T) {
		deal, _ := NewDealFromLotPublished("auction_1", "seller_1", createTestProductSnapshot(), 1000)
		deal.status = DealStatusConfirmed

		err := deal.Validate()
		if err != ErrCustomerIDRequired {
			t.Errorf("want error %v, got %v", ErrCustomerIDRequired, err)
		}
	})
}
