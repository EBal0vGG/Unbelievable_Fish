# Deal Domain

Deal — агрегат домена сделок, фиксирующий факт совершения сделки после аукциона.

## Конструкторы
- `NewDealFromAuctionWon()` — создание сделки при выигрыше аукциона (покупатель + цена + снапшот продукта)
- `NewDealFromProjection()` — создание сделки из проекции (альтернативный вариант)

## Операции
- `Confirm()` — подтверждение сделки (PENDING → CONFIRMED)
- `PrepareContract()` — подготовка контракта
- `SignContract()` — подписание контракта
- `RequestPayment()` — запрос оплаты
- `MarkAsPaid()` — фиксация оплаты
- `RequestShipment()` — запрос доставки
- `MarkAsShipped()` — фиксация отправки
- `Complete()` — завершение сделки
- `Cancel()` — отмена
- `UpdatePrice()` — обновление цены (только в PENDING)

## Статусы
```
PENDING → CONFIRMED → CONTRACT_PREPARED → CONTRACT_SIGNED → 
PAYMENT_REQUESTED → PAID → SHIPMENT_REQUESTED → SHIPPED → COMPLETED
↘ CANCELLED
```

### Геттеры
`ID()`, `CustomerID()`, `SupplierID()`, `AuctionID()`, `UnitPrice()`, 
`Quantity()`, `Status()`, `Type()`, `Contract()`, `ProductName()`, 
`ProductDescription()`, `ProductID()`, `ProductCategory()`, и др.

### События
- `DealCreated` — сделка создана (при AuctionWon)
- `DealConfirmed` — подтверждена
- `ContractPrepared` — контракт подготовлен
- `ContractSigned` — контракт подписан
- `PaymentRequested` — запрос оплаты
- `DealPaid` — оплачена
- `ShipmentRequested` — запрос доставки
- `DealShipped` — отправлена
- `DealCompleted` — завершена
- `DealCancelled` — отменена
- `PriceUpdated` — цена обновлена

### Проекция сделки
- `DealProjection` — read model, создаётся из `LotPublished`
- Хранит снапшот продукта из Catalog Domain
- Превращается в сделку при `AuctionWon`

### Factory
- `Factory.CreateFromProjection()` — создаёт сделку из проекции

---
