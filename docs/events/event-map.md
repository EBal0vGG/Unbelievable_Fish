
# Event-Driven Domain Interaction Map

## Базовый принцип

> **Домен публикует события о своих бизнес-фактах.
> Другие домены реагируют на них, если это их бизнес-интерес.**

> **Если событие другого домена необходимо, чтобы агрегат вообще появился — подписка обязательна.**

> **Междоменные события — integration events.
> Они могут и должны нести достаточно данных для автономной работы consumer’ов.**

---

# Trading Domain (A) — Торги

**Источник истины по:**

* состоянию аукциона
* правилам ставок
* определению победителя

**НЕ знает ничего о товарах, продуктах, UI, пользователях.**

---

## Публикует события

| Event              | Когда                   | Payload                                          |
| ------------------ | ----------------------- | ------------------------------------------------ |
| `AuctionCreated`   | Аукцион создан          | `auction_id`                                     |
| `AuctionPublished` | Аукцион опубликован     | `auction_id`                                     |
| `BidPlaced`        | Принята валидная ставка | `auction_id`, `amount`, `placed_at`              |
| `AuctionClosed`    | Аукцион закрыт          | `auction_id`                                     |
| `AuctionCancelled` | Аукцион отменён         | `auction_id`                                     |
| `AuctionWon`       | Определён победитель    | `auction_id`, `winner_company_id`, `final_price` |


`BidderCompanyID` **передаётся только если downstream реально нужен**
(например, для уведомлений, но не обязательно для Catalog).

---

## Подписывается на события

| Event          | Зачем                                      | Статус |
| -------------- | ------------------------------------------ | ------ |
| `LotPublished` | **создать аукцион под опубликованный лот** | обяз.  |

**Ключевой момент**

Trading **НЕ имеет своего API “CreateAuction” как бизнес-инициативы**.
Аукцион существует **только потому, что был опубликован лот в Catalog**.

Trading реагирует на `LotPublished` и:

* создаёт Auction
* сохраняет `auction_id`
* публикует `AuctionCreated`

---

# Catalog Domain (B) — Каталог

**Источник истины по:**

* товарам
* лотам
* продавцам
* отображению данных для UI

Catalog — **read-model для сайта**.

---

## Публикует события

### `LotPublished` (integration event)

**Событие намеренно “тяжёлое”**

Оно должно позволить downstream-доменам **не ходить в Catalog синхронно**.

**Payload (пример):**

```
LotPublished {
  lot_id
  auction_id
  seller_company_id

  product: {
    product_id
    title
    description
    category
    weight
    volume
    origin_country
  }

  pricing: {
    start_price
    currency
  }
}
```

---

## Подписывается на события

| Event              | Зачем                              | Статус |
| ------------------ | ---------------------------------- | ------ |
| `AuctionCreated`   | связать лот ↔ аукцион (если нужно) | опц    |
| `AuctionPublished` | показать аукцион в каталоге        | да     |
| `BidPlaced`        | обновить текущую цену              | да     |
| `AuctionClosed`    | пометить лот закрытым              | да     |
| `AuctionCancelled` | убрать лот                         | да     |
| `AuctionWon`       | отобразить победу                  | да     |


Catalog **НЕ управляет аукционом**,
он **реагирует и отображает**.

---

# Deal Domain (C) — Сделки

**Источник истины по:**

* факту сделки
* цене сделки
* сторонам сделки

Deal **НЕ знает**:

* как проходили торги
* какие были ставки
* как устроен каталог

---

## Публикует события

| Event             | Когда                | Payload                                        |
| ----------------- | -------------------- | ---------------------------------------------- |
| `DealCreated`     | Сделка зафиксирована | `deal_id`, `auction_id`, `product_id`, `price` |
| `ContractCreated` | Контракт создан      | `deal_id`, `contract_id`                       |

---

## Подписывается на события

| Event          | Зачем                                             |
| -------------- | ------------------------------------------------- |
| `LotPublished` | **создать проекцию AuctionID → Product snapshot** |
| `AuctionWon`   | **создать сделку по выигранному аукциону**        |


Deal:

* **не вызывает Catalog**
* **не запрашивает Product по API**
* всё получает **через события**

---

# Notifications / External Integrations (stub)

## Notifications

| Event         | Зачем                 |
| ------------- | --------------------- |
| `AuctionWon`  | уведомить победителя  |
| `BidPlaced`   | обновить UI (будущее) |
| `DealCreated` | уведомить продавца    |

---

# Общая карта событий

```
Catalog
  └─ LotPublished
        ↓
   Trading
     └─ AuctionCreated
     └─ AuctionPublished
           ↓
        Catalog (UI)
           ↓
        Users place bids
           ↓
        Trading
          └─ BidPlaced
               ↓
        Catalog (price update)
               ↓
        Trading closes auction
          └─ AuctionWon
               ↓
        Deal (create deal)
               ↓
        Notifications
```

---

# Основные сценарии

---

## Сценарий 1: Публикация лота → создание аукциона

```
User
 → Catalog API
   → PublishLot UseCase
     → Lot.Publish()
       → LotPublished
         → EventBus
           → Trading
             → CreateAuction (reaction)
               → AuctionCreated
```


Trading **не дергает API**.
Аукцион появляется **только через событие**.

---

## Сценарий 2: Пользователь делает ставку

```
User
 → Web
   → Trading API (POST /auctions/{id}/bids)
     → PlaceBid UseCase
       → Auction.PlaceBid()
         → BidPlaced
           → EventBus
             → Catalog (обновить цену)
             → Notifications
```

Если ставка меньше текущей:

* домен возвращает ошибку
* **event не публикуется**
* ошибка возвращается клиенту

---

## Сценарий 3: Закрытие аукциона

```
User / Scheduler
 → Trading API
   → CloseAuction
     → Auction.Close()
       → AuctionClosed
       → AuctionWon (если есть ставки)
         → EventBus
           → Catalog
           → Deal
           → Notifications
```

---

## Сценарий 4: Создание сделки

```
EventBus
 → Deal receives AuctionWon
   → Deal UseCase
     → find AuctionLotProjection
     → Deal.Create()
       → DealCreated
         → EventBus
```

Deal **не вызывает**:

* Trading
* Catalog

---