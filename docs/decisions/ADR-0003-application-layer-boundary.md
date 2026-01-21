# ADR-0003: Application Layer Boundary

## Context
The trading domain is implemented as a strong and independent domain model.
Business rules, invariants, and state transitions are fully encapsulated
inside domain aggregates.

With the introduction of domain events as first-class citizens, the system
requires a dedicated layer responsible for:
- orchestrating use cases
- coordinating interaction with the domain
- handling persistence and side effects
- remaining free of business logic

A clear boundary is needed between:
- domain decision-making
- application orchestration
- infrastructure concerns

---

Торговая доменная область реализована как сильная и независимая доменная модель.
Бизнес-правила, инварианты и переходы состояний полностью инкапсулированы
внутри доменных агрегатов.

После введения доменных событий как first-class citizens системе требуется
выделенный слой, отвечающий за:
- оркестрацию сценариев использования (use cases)
- координацию взаимодействия с доменом
- работу с персистентностью и побочными эффектами
- отсутствие бизнес-логики

Необходима чёткая граница между:
- принятием доменных решений
- оркестрацией приложения
- инфраструктурными деталями

## Decision
- Introduce an explicit Application Layer responsible for use case orchestration
- Model application behavior as explicit use cases (e.g. CreateAuction, PublishAuction, PlaceBid, CloseAuction)
- Application layer invokes domain aggregates and reacts to returned domain events
- Business logic and invariants remain exclusively inside the domain layer
- Introduce repositories as interfaces to load and persist aggregate roots
- Repository interfaces are defined at the application boundary
- Repository implementations belong to the infrastructure layer
- Application layer coordinates persistence and event handling but does not interpret business rules

---

- Вводится явный слой Application Layer, отвечающий за оркестрацию сценариев использования
- Поведение приложения моделируется через use cases (например: CreateAuction, PublishAuction, PlaceBid, CloseAuction)
- Слой приложения вызывает методы агрегатов и реагирует на возвращаемые доменные события
- Бизнес-логика и инварианты остаются исключительно внутри доменного слоя
- Репозитории вводятся как интерфейсы для загрузки и сохранения агрегатов
- Интерфейсы репозиториев определяются на границе application layer
- Реализации репозиториев относятся к инфраструктурному слою
- Слой приложения координирует персистентность и обработку событий, но не интерпретирует бизнес-правила

## Consequences
- Domain model remains pure, focused, and independent of infrastructure
- Application layer becomes the single entry point for executing business use cases
- Use cases explicitly define system behavior and intent
- Persistence concerns are abstracted behind repository interfaces
- Domain aggregates are always loaded and saved as consistency boundaries
- Application layer becomes the natural place for transactions and event publishing
- System architecture becomes easier to reason about, test, and evolve

---

- Доменная модель остаётся чистой, сфокусированной и независимой от инфраструктуры
- Application layer становится единой точкой входа для выполнения бизнес-сценариев
- Use cases явно определяют поведение системы и бизнес-намерения
- Детали персистентности скрыты за интерфейсами репозиториев
- Доменные агрегаты всегда загружаются и сохраняются как границы согласованности
- Слой приложения становится естественным местом для управления транзакциями и публикацией событий
- Архитектура системы упрощает понимание, тестирование и дальнейшее развитие
