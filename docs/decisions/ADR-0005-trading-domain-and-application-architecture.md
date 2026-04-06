# ADR-0005: Trading Domain and Application Architecture Updates

## Context
The Trading bounded context has evolved from a simple domain model into a
structured architecture with explicit aggregates, state transitions, domain
events, application use cases, and HTTP boundary contracts.

To keep the system consistent and testable, the architecture must clearly
define:
- aggregate boundaries and invariants
- explicit state transitions (FSM)
- domain events as first-class contract
- application layer orchestration without business logic
- HTTP adapter responsibilities and error mapping
- input ownership (identity headers, AuctionID generation, LotID linkage)
- time-based auction rules and bid history

---

Trading bounded context перешёл от простой доменной модели к архитектуре с
явными агрегатами, переходами состояний, доменными событиями, use case’ами
application layer и HTTP-контрактом.

Чтобы система оставалась согласованной и тестируемой, архитектура должна
зафиксировать:
- границы агрегатов и инварианты
- явную FSM-модель переходов
- доменные события как контракт
- orchestration в application layer без бизнес-логики
- ответственность HTTP-адаптера и маппинг ошибок
- владение входными данными (заголовки идентификации, генерация AuctionID, связь с LotID)
- временные правила аукциона и историю ставок

## Decision
- Auction is a full aggregate with private state, enforced invariants, and FSM transitions
- Domain events are first-class: aggregate methods return events as business facts
- Bid is a value object with factory validation and immutable fields
- Application layer orchestrates use cases: Load → domain call → Save → Publish
- AuctionID is generated in the application layer and passed into the domain
- Auction stores LotID for linkage to catalog/lot
- HTTP adapter parses requests, passes headers/context, maps errors via a contract
- Error mapping is centralized and reusable outside HTTP
- Auction rules include time-based start/end, late-bid extension, and bid floor
- Bid history is preserved in the aggregate (repository handles persistence)

---

- Auction — полноценный агрегат с приватным состоянием, инвариантами и FSM
- Доменные события — first-class контракт: методы агрегата возвращают события
- Bid — value object с фабрикой, валидацией и неизменяемыми полями
- Application layer оркестрирует use case: Load → домен → Save → Publish
- AuctionID генерируется в application layer и передаётся в домен
- Auction хранит LotID для связи с лотом/каталогом
- HTTP-адаптер разбирает запросы, передаёт контекст, маппит ошибки по контракту
- Маппинг ошибок централизован и переиспользуем вне HTTP
- В правилах аукциона учтены старт/конец, продление при поздней ставке и минимальная цена
- История ставок сохраняется в агрегате (репозиторий отвечает за хранение)

## Consequences
- The Trading architecture is explicit, deterministic, and testable end-to-end
- Domain and application logic are isolated from transport/infrastructure
- External clients can rely on stable HTTP and event contracts
- Cross-context integration is enabled via events and LotID linkage
- Error handling is consistent across adapters
- Time-based auction behavior is enforced by the domain, not handlers

---

- Архитектура Trading стала явной, детерминированной и тестируемой
- Домен и application layer изолированы от транспорта и инфраструктуры
- Внешние клиенты опираются на стабильные HTTP и event-контракты
- Междоменные интеграции возможны через события и связь по LotID
- Обработка ошибок едина для разных адаптеров
+- Временное поведение аукциона гарантируется доменом, а не хендлерами
