# Deal Domain API  
Deal является агрегатом домена сделок, фиксирующим факт совершения сделки после аукциона. Создается только через конструкторы на основе событий.

## Конструкторы
- `NewDealFromLotPublished()` - драфт при публикации лота (только продавец)
- `CompleteDealFromAuctionWon()` - завершение при выигрыше (покупатель + цена)

## Операции  
- `Confirm()` - подтверждение сделки (DRAFTED/PENDING → CONFIRMED)
- `RequestPayment()` - создание контракта для оплаты
- `MarkAsPaid()` - фиксация оплаты
- `RequestShipment()` - запрос доставки
- `MarkAsShipped()` - фиксация отправки
- `Complete()` - завершение сделки
- `Cancel()` - отмена

## Статусы  
- `DRAFTED` - создан при LotPublished (нет покупателя)
- `PENDING` - заполнен при AuctionWon (есть покупатель)
- `CONFIRMED` - подтвержден
- `PAYMENT_REQUESTED` - запрос оплаты
- `PAID` - оплачено
- `SHIPMENT_REQUESTED` - запрос доставки
- `SHIPPED` - отправлено
- `COMPLETED` - доставлено
- `CANCELLED` - отменено

## Геттеры  
- `ID()`, `CustomerID()`, `SupplierID()`, `AuctionID()`
- `UnitPrice()`, `Quantity()`, `Status()`, `Type()`
- `ProductName()`, `ProductDescription()`, `ProductID()`

## События (application layer)  
- `DealCreated` (при создании драфта)
- `DealConfirmed` (при подтверждении)
- `PaymentRequested` (при запросе оплаты)
- `DealPaid` (при оплате)
- `ShipmentRequested` (при запросе доставки)
- `DealShipped` (при отправке)
- `DealCompleted` (при завершении)
- `DealCancelled` (при отмене)

**Ошибки**  
- `ErrAuctionIDRequired` - требуется auctionID
- `ErrSellerCompanyRequired` - требуется продавец
- `ErrProductNameRequired` - требуется название продукта
- `ErrWinnerCompanyRequired` - требуется победитель
- `ErrFinalPricePositive` - цена должна быть > 0
- `ErrOnlyDraftCanBeCompleted` - можно завершить только драфт
- `ErrCannotConfirmDeal` - нельзя подтвердить в текущем статусе
- `ErrCannotRequestPayment` - нельзя запросить оплату
- `ErrCannotMarkAsPaid` - нельзя отметить как оплаченную
- `ErrCannotRequestShipment` - нельзя запросить доставку
- `ErrCannotMarkAsShipped` - нельзя отметить как отправленную
- `ErrCannotCompleteDeal` - нельзя завершить сделку
- `ErrCannotCancelDeal` - нельзя отменить сделку
- `ErrPriceMustBePositive` - цена должна быть положительной