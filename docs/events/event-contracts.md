# Event Contracts (Catalog, Auction, Deal)

Unified event catalog for the future EventBus. Sources are current code
definitions in:
- `internal/catalog/domain/events.go`
- `internal/trading/auction/events.go`
- `internal/deals/deal/events.go`

---

## Catalog → Trading

### LotPublished
Producer: Catalog  
Payload:
- `lot_id`
- `auction_id`
- `seller_company_id`
- `product_id`
- `product` (snapshot: `fish_id`, `weight`, `unit`, `size`, `processing_type`)
- `start_price`
- `status`

### LotUnpublished
Producer: Catalog  
Payload:
- `lot_id`
- `status`

### LotAuctionLinked
Producer: Catalog  
Payload:
- `lot_id`
- `auction_id`

### LotPriceUpdated
Producer: Catalog  
Payload:
- `lot_id`
- `auction_id`
- `current_price`
- `status`

### LotClosed
Producer: Catalog  
Payload:
- `lot_id`
- `final_price`
- `status`

### LotCreated
Producer: Catalog  
Payload:
- `lot_id`
- `product_id`
- `seller_company_id`
- `photo`
- `quantity`
- `status`

### ProductCreated
Producer: Catalog  
Payload:
- `product_id`
- `fish_id`
- `weight`
- `unit`
- `size`
- `processing_type`
- `status`

### ProductUpdated
Producer: Catalog  
Payload:
- `product_id`
- `fish_id`
- `weight`
- `unit`
- `size`
- `processing_type`
- `status`

### ProductPublished
Producer: Catalog  
Payload:
- `product_id`
- `status`

### ProductUnpublished
Producer: Catalog  
Payload:
- `product_id`
- `status`

---

## Auction → Deal

### AuctionPublished
Producer: Trading/Auction  
Payload:
- `auction_id`

### BidPlaced
Producer: Trading/Auction  
Payload:
- `auction_id`
- `bidder_company_id`
- `amount`
- `placed_at`
- `new_end_at`

### AuctionClosed
Producer: Trading/Auction  
Payload:
- `auction_id`

### AuctionWon
Producer: Trading/Auction  
Payload:
- `auction_id`
- `lot_id`
- `winner_company_id[]`
- `final_price`

### AuctionCancelled
Producer: Trading/Auction  
Payload:
- `auction_id`

---

## Deal → Catalog / Notifications / Payment

### DealCreated
Producer: Deal  
Payload:
- `deal_id`
- `auction_id`
- `customer_id`
- `supplier_id`
- `product_snapshot` (snapshot: `product_id`, `name`, `description`, `category`, `weight`, `volume`, `origin_country`)
- `final_price`
- `created_at`

### DealConfirmed
Producer: Deal  
Payload:
- `deal_id`
- `confirmed_at`

### ContractPrepared
Producer: Deal  
Payload:
- `deal_id`
- `contract_number`
- `prepared_at`
- `document_url`

### ContractSigned
Producer: Deal  
Payload:
- `deal_id`
- `contract_number`
- `signed_at`
- `signed_by`
- `signature_ref`

### PaymentRequested
Producer: Deal  
Payload:
- `deal_id`
- `total_amount`
- `invoice_number`
- `due_date`
- `requested_at`

### DealPaid
Producer: Deal  
Payload:
- `deal_id`
- `payment_id`
- `payment_type`
- `paid_at`

### ShipmentRequested
Producer: Deal  
Payload:
- `deal_id`
- `requested_at`

### DealShipped
Producer: Deal  
Payload:
- `deal_id`
- `tracking_number`
- `carrier`
- `shipped_at`

### DealCompleted
Producer: Deal  
Payload:
- `deal_id`
- `completed_at`

### DealCancelled
Producer: Deal  
Payload:
- `deal_id`
- `reason`
- `cancelled_by`
- `cancelled_at`

### PriceUpdated
Producer: Deal  
Payload:
- `deal_id`
- `old_price`
- `new_price`
- `updated_by`
- `updated_at`

