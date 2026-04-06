# Catalog Events

Источник: [events.go](/Users/alitagiev/Desktop/fish/internal/catalog/domain/events.go)

## Event Interface

```go
type Event interface {
	isCatalogEvent()
}
```

## Product Events

### ProductCreated

Поля:
- `ProductID`
- `FishID`
- `Weight`
- `Unit`
- `Size`
- `ProcessingType`
- `Status`

Генерация:
- `NewProduct(...)`

### ProductUpdated

Поля:
- `ProductID`
- `FishID`
- `Weight`
- `Unit`
- `Size`
- `ProcessingType`
- `Status`

Генерация:
- `Product.Update(...)`

### ProductPublished

Поля:
- `ProductID`
- `Status`

Генерация:
- `Product.Publish()`

### ProductUnpublished

Поля:
- `ProductID`
- `Status`

Генерация:
- `Product.Unpublish()`

## Lot Events

### LotCreated

Поля:
- `LotID`
- `ProductID`
- `SellerCompanyID`
- `Photo`
- `Quantity`
- `Status`

Генерация:
- `NewLot(...)`

### LotPublished

Поля:
- `LotID`
- `AuctionID`
- `SellerCompanyID`
- `ProductID`
- `Product`
- `StartPrice`
- `Status`

Генерация:
- `Lot.Publish(...)`

### LotUnpublished

Поля:
- `LotID`
- `Status`

Генерация:
- `Lot.Unpublish()`

### LotClosed

Поля:
- `LotID`
- `FinalPrice`
- `Status`

Генерация:
- `Lot.Close(...)`

### LotPriceUpdated

Поля:
- `LotID`
- `AuctionID`
- `CurrentPrice`
- `Status`

Генерация:
- `Lot.UpdateCurrentPrice(...)`

### LotAuctionLinked

Поля:
- `LotID`
- `AuctionID`

Статус:
- объявлен
- сейчас не эмитится

## ProductSnapshot

Используется внутри `LotPublished`.

Поля:
- `ProductID`
- `Name`
- `Weight`
- `Unit`
- `Size`
- `ProcessingType`

## Outbox

События сохраняются в outbox через:
- [service.go](/Users/alitagiev/Desktop/fish/internal/catalog/app/service.go)
- [outbox_repository.go](/Users/alitagiev/Desktop/fish/internal/catalog/postgres/outbox_repository.go)

Для каждого события в `outbox_messages` пишутся:
- `id`
- `event_id`
- `event_type`
- `aggregate_id`
- `payload`
- `occurred_at`
- `created_at`
- `published_at`
