**Scope & Sources**  
Проанализирован весь код в репозитории: `domain/catalog`, `internal/catalog/app`, `migrations`, `README.md`. В ветке `origin/ebal0v` доступен только `docs/events/event-map.md`; `er-diagram.puml/svg` в репозитории отсутствуют. Анализ ограничен тем, что реально есть в коде.

**Project Map**  
- `domain/catalog`: доменная модель Catalog (Fish, Product, Lot), события, FSM, ошибки, тесты.  
- `internal/catalog/app`: application/use case слой Catalog, порты репозиториев, тесты.  
- `migrations`: минимальные DDL/seed справочников (fish, units, processing_types).  
- Инфраструктура (реализации репозиториев, outbox, брокер, HTTP handlers/consumers) отсутствует.  

---

**Bounded Contexts**  
- **Catalog**: единственный реализованный домен.  
- **Trading** и **Deal**: в коде отсутствуют, присутствуют только в `README.md` и `docs/events/event-map.md`.  

---

**Catalog Domain (domain/catalog)**

**Fish (Entity)**  
- Поля: `fishID`, `name`, `description`.  
- Инварианты: `fishID` и `name` не пустые.  
- Методы:  
  - `NewFish` — создаёт Fish, валидирует идентификатор и имя.  
  - `Update` — меняет `name/description`, имя не пустое.  
- Событий не генерирует.

**Product (Aggregate Root)**  
- Поля: `productID`, `fishID`, `weight float64`, `size`, `unit string`, `processingType ProcessingType`, `status ProductStatus`.  
- Инварианты:  
  - `productID`, `fishID` не пустые.  
  - `unit` не пустой.  
  - `processingType` не пустой (только проверка на пустую строку).  
  - `weight > 0`.  
- Методы:  
  - `NewProduct` — создаёт Product, статус `DRAFT`, генерирует `ProductCreated`.  
  - `Update` — только в `DRAFT`, валидирует те же базовые поля, генерирует `ProductUpdated`.  
  - `Publish` — только из `DRAFT`, генерирует `ProductPublished`.  
  - `Unpublish` — только из `PUBLISHED`, возвращает в `DRAFT`, генерирует `ProductUnpublished`.  

**Lot (Aggregate Root)**  
- Поля:  
  - идентичность: `lotID`, `productID`, `sellerCompanyID`, `auctionID`.  
  - атрибуты: `photo`, `quantity float64`.  
  - цены: `startPrice`, `curPrice`, `finalPrice` (int64, рубли).  
  - статус: `LotStatus`.  
  - расписание: `auctionSchedule` (startsAt через `Instant`).  
- Инварианты:  
  - `lotID/productID/sellerCompanyID` не пустые.  
  - `quantity > 0`.  
  - `startPrice > 0`.  
  - `auctionSchedule != nil`.  
- Методы:  
  - `NewLot` — создаёт Lot в `DRAFT`, `curPrice` и `finalPrice` равны `startPrice`, генерирует `LotCreated`.  
  - `AssignAuctionID` — только в `DRAFT`, только один раз, без событий.  
  - `Publish(productIsPublished bool, product ProductSnapshot)` — только если переход разрешён и Product опубликован, требует `auctionID`, генерирует “тяжёлый” `LotPublished`.  
  - `Unpublish` — переводит в `CANCELLED`, генерирует `LotUnpublished`.  
  - `Close(finalPrice)` — переводит в `CLOSED`, `finalPrice > 0`, генерирует `LotClosed`.  
  - `UpdateCurrentPrice(amount)` — только в `PUBLISHED`, `amount > 0`, обновляет `curPrice`, событий не генерирует.  

**Value Objects / Types**  
- `ProcessingType`, `Unit`, `PackagingType` — типы-обёртки string без валидации значений.  
- `AuctionSchedule` / `Instant` — хранят дату старта аукциона (`UTC`).

---

**Catalog Domain Events (domain/catalog/events.go)**  
- `ProductCreated`, `ProductUpdated`, `ProductPublished`, `ProductUnpublished`.  
- `LotCreated`, `LotPublished`, `LotUnpublished`, `LotClosed`.  
- `LotAuctionLinked` — существует, но в коде не используется.  
- `LotPublished` является интеграционным “тяжёлым” событием и содержит: `lot_id`, `auction_id`, `seller_company_id`, `product_id`, `product snapshot`, `start_price`, `status`.  
- `ProductSnapshot` содержит: `fish_id`, `weight`, `unit`, `size`, `processing_type`.  

Все события реализуют `Event` через `isCatalogEvent()`.

---

**FSM & Invariants (domain/catalog/state.go, transitions.go)**  
- ProductStatus: `DRAFT`, `PUBLISHED`, `UNPUBLISHED`.  
- LotStatus: `DRAFT`, `PUBLISHED`, `CLOSED`, `CANCELLED`.  
- `lotTransitions` используется в `Lot.canTransition`.  
- `productTransitions` есть, но `Product` методы на неё не опираются.  

---

**Application Layer (internal/catalog/app)**

**Порты**  
- Репозитории: `FishRepository`, `ProductRepository`, `LotRepository`, `UnitRepository`, `ProcessingTypeRepository`.  
- `OutboxRepository` для событий.  
- `TransactionManager` с `WithinTx`.

**CatalogService (use cases)**  
- `CreateFish`, `UpdateFish`.  
- `CreateProduct`, `UpdateProduct`, `PublishProduct`, `UnpublishProduct`.  
- `CreateLot`, `AssignAuctionID`, `PublishLot`, `UnpublishLot`, `CloseLot`.  
- Trading consumers: `HandleAuctionWon`, `HandleBidPlaced`, `HandleAuctionClosed`, `HandleAuctionCancelled`.  

**Общие правила use case**  
- Везде есть `WithinTx`.  
- Load из репозитория → вызов домена → save агрегата → save events в outbox.  
- Валидация справочников fish/unit/processingType происходит в use case (`validateProductRefs`).

---

**Use Cases — детальный разбор**

**CreateFish**  
- Вход: `CreateFishCommand`.  
- Действия: `catalog.NewFish` → `fishRepo.Save`.  
- События: нет.

**UpdateFish**  
- Load: `fishRepo.Get`.  
- Домен: `fish.Update`.  
- Save: `fishRepo.Save`.  
- События: нет.

**CreateProduct**  
- Validate refs: `fishRepo.Exists`, `unitRepo.Exists`, `processingTypeRepo.Exists`.  
- Домен: `catalog.NewProduct`.  
- Save: `productRepo.Save`.  
- Outbox: события `ProductCreated`.

**UpdateProduct**  
- Load: `productRepo.Get`.  
- Validate refs: через репозитории.  
- Домен: `product.Update`.  
- Save: `productRepo.Save`.  
- Outbox: `ProductUpdated`.

**PublishProduct / UnpublishProduct**  
- Load: `productRepo.Get`.  
- Домен: `Publish` / `Unpublish`.  
- Save: `productRepo.Save`.  
- Outbox: `ProductPublished` / `ProductUnpublished`.

**CreateLot**  
- Load: `productRepo.Get` для проверки существования.  
- Домен: `catalog.NewLot` (с `AuctionSchedule`).  
- Save: `lotRepo.Save`.  
- Outbox: `LotCreated`.

**AssignAuctionID**  
- Load: `lotRepo.Get`.  
- Домен: `lot.AssignAuctionID`.  
- Save: `lotRepo.Save`.  
- Outbox: нет событий.

**PublishLot**  
- Load: `lotRepo.Get` + `productRepo.Get`.  
- Домен: `lot.Publish(productIsPublished, ProductSnapshot)`.  
- Save: `lotRepo.Save`.  
- Outbox: `LotPublished`.

**UnpublishLot**  
- Load: `lotRepo.Get`.  
- Домен: `lot.Unpublish`.  
- Save: `lotRepo.Save`.  
- Outbox: `LotUnpublished`.

**CloseLot**  
- Load: `lotRepo.Get`.  
- Домен: `lot.Close`.  
- Save: `lotRepo.Save`.  
- Outbox: `LotClosed`.

**HandleAuctionWon**  
- Load: `lotRepo.GetByAuctionID`.  
- Домен: `lot.Close(finalPrice)` если не закрыт.  
- Save + Outbox: `LotClosed`.  

**HandleBidPlaced**  
- Load: `lotRepo.GetByAuctionID`.  
- Домен: `lot.UpdateCurrentPrice(amount)`.  
- Save: `lotRepo.Save`.  
- Outbox: нет событий.  

**HandleAuctionClosed**  
- Load: `lotRepo.GetByAuctionID`.  
- Домен: `lot.Close(lot.CurPrice())`.  
- Save + Outbox: `LotClosed`.  

**HandleAuctionCancelled**  
- Load: `lotRepo.GetByAuctionID`.  
- Домен: `lot.Unpublish()`.  
- Save + Outbox: `LotUnpublished`.  

---

**Flows по сценариям**

**Создание Fish**  
- Инициатор: UI/админ (HTTP, предположительно).  
- Use case: `CreateFish`.  
- Домен: `NewFish`.  
- События: нет.  
- Хранилище: `FishRepository`.

**Создание Product**  
- Инициатор: UI/админ (HTTP).  
- Use case: `CreateProduct`.  
- Проверки: fish/unit/processingType в справочниках через репозитории.  
- Домен: `NewProduct`.  
- События: `ProductCreated` → outbox.  

**Публикация Product**  
- Инициатор: UI/админ (HTTP).  
- Use case: `PublishProduct`.  
- Домен: `Publish`.  
- События: `ProductPublished` → outbox.  

**Создание Lot**  
- Инициатор: UI/админ (HTTP).  
- Use case: `CreateLot`.  
- Домен: `NewLot`.  
- События: `LotCreated` → outbox.

**Назначение auctionID**  
- Инициатор: consumer от Trading (AuctionCreated) или админ (не реализовано).  
- Use case: `AssignAuctionID`.  
- Домен: `AssignAuctionID`.  
- События: нет.  

**Публикация Lot**  
- Инициатор: UI/админ (HTTP).  
- Use case: `PublishLot`.  
- Домен: `Publish(productIsPublished, ProductSnapshot)`.  
- События: интеграционное `LotPublished` → outbox.  

**AuctionWon (Trading → Catalog)**  
- Transport: event consumer (предполагается).  
- Use case: `HandleAuctionWon`.  
- Домен: `Close(finalPrice)`.  
- События: `LotClosed` → outbox.  

**BidPlaced (Trading → Catalog)**  
- Use case: `HandleBidPlaced`.  
- Домен: `UpdateCurrentPrice`.  
- События: нет (только обновление состояния).

**AuctionClosed / AuctionCancelled (Trading → Catalog)**  
- `HandleAuctionClosed`: закрывает лот по `curPrice`.  
- `HandleAuctionCancelled`: переводит лот в `CANCELLED`.  
- События: `LotClosed` или `LotUnpublished`.

**Deal / Projections**  
- Код отсутствует. В проекте нет Deal‑домена и проекций.  

---

**Events: кто публикует и кто потребляет**

**Catalog publishes**  
- `ProductCreated/Updated/Published/Unpublished` — доменные события.  
- `LotCreated/Published/Unpublished/Closed` — доменные и потенциально интеграционные.  
- `LotPublished` — явно интеграционное (heavy payload).  

**Catalog consumes (через use cases)**  
- Trading: `AuctionWon`, `BidPlaced`, `AuctionClosed`, `AuctionCancelled`.  
- Trading: `AuctionCreated`, `AuctionPublished` — явных обработчиков нет (см. риски).  

---

**Tests**

**Domain tests**  
- `fish_test.go`: валидация и update.  
- `product_test.go`: валидация, publish/unpublish, update‑only‑in‑draft.  
- `lot_test.go`: publish constraints, assign auction id, unpublish, close, price/quantity/photo validation.

**Use case tests**  
- Проверяют:  
  - CreateProduct с несуществующими справочниками → ошибки.  
  - UpdateProduct с несуществующими справочниками → ошибки.  
  - PublishLot требует опубликованный Product.  
  - PublishLot требует `auctionID`.  
  - CloseLot пишет `LotClosed` в outbox.  
  - HandleAuctionWon работает по `auctionID`.  

Покрытие есть, но отсутствуют тесты на обработчики `BidPlaced`, `AuctionClosed`, `AuctionCancelled`.

---

**Слои архитектуры**

**Domain layer**  
- Чистый домен: без инфраструктуры, HTTP, DB, broker.  
- Агрегаты возвращают `[]Event, error`.  

**Application layer**  
- Orchestration, валидация справочников через репозитории.  
- Tx boundary через `TransactionManager`.  
- Outbox через `OutboxRepository`.  

**Infrastructure layer**  
- Отсутствует (нет реализаций репозиториев, outbox, consumers, HTTP).  
- Есть только миграции справочников как hint.

**Transport layer**  
- Отсутствует. Из README и docs видно, что предполагается HTTP для команд и consumer для событий.

---

## Итоговые секции (как просили)

**A. Как устроен проект в целом**  
Проект содержит только Catalog: доменная модель и application layer. Trading/Deal отсутствуют. Справочники предполагаются в БД (есть миграции). Взаимодействие с внешними доменами оформлено DTO и use case‑обработчиками.

**B. Какой flow у данных и событий**  
Command‑flow: HTTP → use case → домен → save → outbox.  
Event‑flow: Trading event → consumer → use case → домен → save → outbox.  
В коде реализованы обработчики для `AuctionWon`, `BidPlaced`, `AuctionClosed`, `AuctionCancelled`.

**C. Где какая бизнес‑логика находится**  
- Domain: инварианты, FSM, факты (events).  
- Application: orchestration, проверка справочников, транзакции, outbox.  
- Infrastructure/transport: отсутствуют, ожидаются внешние реализации.

**D. Какие use cases есть и за что отвечают**  
Полный набор Catalog use cases: `Create/Update Fish`, `Create/Update/Publish/Unpublish Product`, `Create/AssignAuctionID/Publish/Unpublish/Close Lot`, `HandleAuctionWon/BidPlaced/AuctionClosed/AuctionCancelled`.  

**E. Противоречия / слабые места / архитектурные риски**  
1. `domain/catalog/README.md` устарел: описывает `LotSold/MarkSold/SOLD`, которых в коде нет.  
2. `ProductStatusUnpublished` существует, но `Unpublish` возвращает в `DRAFT`; статус `UNPUBLISHED` недостижим.  
3. `productTransitions` объявлен, но нигде не используется (FSM не единообразен).  
4. `LotAuctionLinked` событие объявлено, но нигде не эмитится и не используется.  
5. В `docs/events` ожидаются `AuctionCreated/AuctionPublished` обработчики; в коде их нет.  
6. `LotPublished` payload по docs включает поля (title/description/category/currency), которых в домене нет — невозможно выполнить контракт полностью.  
7. `UpdateCurrentPrice` не генерирует событий; если UI/другие системы должны реагировать на обновление цены, это не реализовано.  
8. Нет инфраструктуры и транспортного слоя, поэтому “use case — единственная точка входа” пока не закреплён кодом (нет handlers/consumers).  

**F. Что уже сделано хорошо**  
- Домен изолирован от инфраструктуры.  
- Use case слой чётко оркестрирует изменения и outbox.  
- Валидация справочников вынесена в application layer.  
- `LotPublished` уже “heavy” и содержит продуктовый snapshot.  
- Тесты на ключевые инварианты домена и основные use cases уже есть.  

---

**go test ./...**  
Проходит: `ok` для `domain/catalog` и `internal/catalog/app`.
