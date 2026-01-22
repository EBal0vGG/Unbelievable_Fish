# Catalog Domain

Catalog is a **supporting domain** responsible for defining **what is being sold**
and **whether it is allowed to be sold** on the platform.

Catalog contains **no trading logic** and acts as a prerequisite for the Trading domain.

The domain models **physical goods only** (fish products and their batches).
Services (logistics, banking, insurance, etc.) are explicitly out of scope.



Catalog — это **поддерживающий домен**, отвечающий за определение:
- **что именно продаётся** на платформе;
- **допустимо ли выставлять это на продажу**.

Catalog **не содержит торговой логики** и является обязательной предпосылкой
для работы домена торгов (Trading).

Домен моделирует **только физические товары** (рыбную продукцию и её партии).
Услуги (логистика, банки, страхование и т.п.) намеренно вынесены за пределы домена.
---

## Aggregates

Catalog consists of two aggregate roots:

- **Product** — product card describing a sellable good
- **Lot** — concrete batch / offer of a product

Each aggregate is the **single source of truth** for its own lifecycle and invariants.

---

## Product

Product represents a **product card** describing a physical fish good.

Product is the single source of truth for:
- product description
- publication readiness
- product lifecycle state

Product is created exclusively via `NewProduct(...)`.

### Lifecycle

DRAFT → PUBLISHED → SOLD
→ CANCELLED

---

### Operations

**NewLot(...)**
- creates a lot in `DRAFT` state
- validates identifiers, quantity and unit
- emits `LotCreated`

**Publish(productIsPublished bool)**
- moves lot from `DRAFT` to `PUBLISHED`
- allowed only if the related Product is published
- emits `LotPublished`

**Unpublish()**
- cancels a published lot
- moves lot from `PUBLISHED` to `CANCELLED`
- emits `LotUnpublished`

**MarkSold(dealID string)**
- marks a published lot as sold
- requires non-empty `dealID`
- moves lot from `PUBLISHED` to `SOLD`
- emits `LotSold`

**Accessors**
- `ID()`, `Status()`, `DealID()` — read-only

---

### Events

- `LotCreated`
- `LotPublished`
- `LotUnpublished`
- `LotSold`

---

### Errors (contract)

- `ErrInvalidIdentifier` — invalid lot, product or seller identifier
- `ErrInvalidQuantity` — quantity must be positive
- `ErrInvalidEnumValue` — invalid unit of measurement
- `ErrProductNotPublished` — lot cannot be published if product is not published
- `ErrInvalidStateTransition` — forbidden lifecycle transition
- `ErrDealIDEmpty` — deal identifier is required when marking as sold

---

## Domain Boundaries

Catalog **does not contain**:
- trading logic
- bids or pricing
- auctions or winners
- logistics or services
- banking or insurance logic
- infrastructure or persistence

Catalog **guarantees**:
- only valid products can be published
- only valid lots can be exposed to the market
- Trading operates on consistent and verified data

---

## Architectural Notes

- Implemented as a pure Go domain model
- No framework or infrastructure dependencies
- Explicit finite state machines (FSM)
- Domain events are first-class citizens
- All invariants are enforced inside aggregates
- Product and Lot are isolated aggregates
- Product–Lot relation is indirect (via identifiers only)

---

## Relation to Trading Domain

Catalog defines **what is allowed to be sold**.  
Trading defines **how it is sold**.

Trading depends on Catalog, but Catalog does not depend on Trading.
