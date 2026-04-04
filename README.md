# РЫБНАЯ БИРЖА

# 1️⃣ Сценарий: запуск (создание + публикация) аукциона

## Создание аукциона (Catalog → Trading)

**Клиент (админ / менеджер)**
↓
**Web (UI / Backoffice)**
↓
**Router Catalog**
↓
**HTTP handler Catalog**
↓
**Catalog Application use case**
↓
**Catalog Domain method**
→ формирует бизнес-факт
→ **AuctionCreated (или AuctionPrepared)**
↓
**Catalog Repository (save)**
↓
**Catalog Event Publisher**
→ публикует `AuctionCreated`

---

## Доставка события

**Event Bus / Message Broker / Outbox relay**
↓
**Trading subscriber (infrastructure adapter)**
→ НЕ HTTP
→ НЕ handler
→ background consumer

---

## Обработка события в Trading

**Trading Event Handler (integration layer)**
↓
**Trading Application use case (HandleAuctionCreated)**
↓
**Trading Domain method (Auction.Create / DraftFromCatalog)**
→ валидирует инварианты Trading
→ формирует `AuctionDrafted`
↓
**Trading Repository (save)**
↓
**Trading Event Publisher**
→ публикует `AuctionDrafted`

---

## Кто получает `AuctionDrafted`

* ❓ *никто* — если это внутренний факт Trading
* 📦 Catalog (если нужно показать статус)
* 📊 Analytics / Audit
* 🧪 Test subscribers

📌 **Важно:**
Trading **не знает**, кто подписан. Он просто публикует факт.

---

# 2️⃣ Сценарий: публикация аукциона (Trading only)

**Клиент**
↓
**Web**
↓
**Trading Router**
↓
**Trading HTTP handler (PublishAuction)**
↓
**Trading Application use case**
↓
**Trading Domain method (Publish)**
→ проверка состояния
→ формирует `AuctionPublished`
↓
**Trading Repository (save)**
↓
**Trading Event Publisher**
→ публикует `AuctionPublished`

---

## Кто получает `AuctionPublished`

* **Catalog** → начинает показывать лот
* **Search / Indexer**
* **Notification service**
* **WebSocket gateway**

📌 HTTP-ответ клиенту **не ждёт**, пока события будут обработаны.

---

# 3️⃣ Сценарий: просмотр аукциона (Query)

**Клиент**
↓
**Web**
↓
**Trading Router**
↓
**Query handler (GetAuction)**
↓
**Read model / Projection / View repo**
↓
**Response**

📌

* **никаких событий**
* **никакого домена**
* **никаких use case команд**

---

# 4️⃣ Сценарий: ставка (Bid)

**Клиент (участник торгов)**
↓
**Web**
↓
**Trading Router**
↓
**PlaceBid HTTP handler**
↓
**Trading Application use case**
↓
**Trading Domain method (PlaceBid)**
→ проверка состояния
→ проверка суммы
→ формирует `BidPlaced`
↓
**Trading Repository (save)**
↓
**Trading Event Publisher**
→ публикует `BidPlaced`

---

## Кто получает `BidPlaced`

* **Catalog** → обновить цену
* **WebSocket / Realtime**
* **Anti-fraud**
* **Analytics**

---

# 5️⃣ Сценарий: закрытие аукциона

**Client / Scheduler / Cron**
↓
**Trading Router / Internal trigger**
↓
**CloseAuction handler**
↓
**Trading Application use case**
↓
**Trading Domain method (Close)**
→ выбирает победителя
→ формирует:

* `AuctionClosed`
* `AuctionWon`
  ↓
  **Trading Repository (save)**
  ↓
  **Trading Event Publisher**

---

## Кто получает события закрытия

* **Catalog** → финальный статус
* **Payments** → резерв / списание
* **Notification** → победителю
* **Legal / Audit**

---

# 6️⃣ Сценарий: отмена аукциона

**Client / Admin**
↓
**Trading Router**
↓
**CancelAuction handler**
↓
**Trading Application use case**
↓
**Trading Domain method (Cancel)**
→ формирует `AuctionCancelled`
↓
**Trading Repository (save)**
↓
**Trading Event Publisher**

---

## Кто получает `AuctionCancelled`

* **Catalog**
* **Notification**
* **Search**

---

# 7️⃣ Ключевые правила

### ❌ Что никогда не происходит

* событие не вызывает HTTP handler
* домен не знает про transport
* сервис не знает, кто подписан
* события не используются как RPC

### ✅ Что всегда происходит

* событие → application use case
* use case → домен
* домен → события
* инфраструктура → доставка

---

# 8️⃣ Если одной фразой

> **HTTP — для людей и UI.
> Events — для систем.
> Use case — единственная точка входа в домен.**
