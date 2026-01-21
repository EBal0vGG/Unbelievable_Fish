# Auction Domain API

Auction is the aggregate root of the trading domain and the only source of
truth for auction state and bidding rules. Create it via `NewAuction(id)`.
---
Аукцион является корневым агрегатом торговой доменной области и единственным
источником истины о состоянии аукциона и правилах приёма ставок.
Создаётся исключительно через конструктор `NewAuction(id)`.

## Operations

- `Publish()` moves the auction from `DRAFT` to `PUBLISHED`.
- `PlaceBid(bid)` accepts a valid bid only in `PUBLISHED` state.
- `Close()` ends the auction:
  - no bids -> `CANCELLED`
  - bids -> `CLOSED` then `WON`
- `Cancel()` cancels an auction in `PUBLISHED` state with no bids.
- `State()`, `Bids()`, `Winner()` are read-only accessors.

## Events returned

- `Publish()` -> `AuctionPublished`
- `PlaceBid()` -> `BidPlaced`
- `Close()` with no bids -> `AuctionCancelled`
- `Close()` with bids -> `AuctionClosed`, `AuctionWon`
- `Cancel()` -> `AuctionCancelled`

## Errors (contract)

- `ErrAuctionCannotBePublished` — аукцион нельзя опубликовать в текущем состоянии.
- `ErrAuctionNotActive` — аукцион не находится в активном состоянии.
- `ErrCannotCloseAuction` — аукцион нельзя закрыть.
- `ErrInvalidStateTransition` — недопустимый переход между состояниями.
- `ErrCannotCancelWithBids` — невозможно отменить аукцион, если уже есть ставки.
- `ErrBidderCompanyIDEmpty` — идентификатор компании-участника пуст.
- `ErrBidAmountNonPositive` — сумма ставки должна быть больше нуля.
- `ErrBidPlacedAtZero` — время размещения ставки не задано.