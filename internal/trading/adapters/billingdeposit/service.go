package billingdeposit

import (
	"context"
	"errors"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

// Service wires billing reserve-deposit into trading's DepositService port (RUB only).
type Service struct {
	create  *billingapp.CreateAccount
	reserve *billingapp.ReserveAuctionDeposit
}

func NewService(create *billingapp.CreateAccount, reserve *billingapp.ReserveAuctionDeposit) *Service {
	return &Service{create: create, reserve: reserve}
}

func (s *Service) ReserveAuctionDeposit(ctx context.Context, companyID, auctionID string, startPrice int64) error {
	if err := s.create.Execute(ctx, companyID); err != nil {
		return err
	}
	err := s.reserve.Execute(ctx, companyID, auctionID, startPrice, wallet.CurrencyRUB)
	if errors.Is(err, billingapp.ErrInsufficientFundsForDeposit) {
		return tradingapp.ErrInsufficientFundsForDeposit
	}
	return err
}
