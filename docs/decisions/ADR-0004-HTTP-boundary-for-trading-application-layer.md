# ADR-0004: HTTP Boundary for Trading Application Layer

## Context
The Trading domain and application layer are implemented as isolated,
framework-agnostic components responsible for business rules and use case
orchestration.

To enable collaboration between teams and allow external clients (web, other
services) to interact with Trading, a clear HTTP boundary must be introduced.

This boundary must:
- expose Trading capabilities via a stable contract
- not leak domain or application internals
- not introduce business logic outside the domain
- serve as an adapter between HTTP and the application layer

---

Торговая доменная модель и слой приложения реализованы как изолированные,
независимые от фреймворков компоненты, отвечающие за бизнес-правила и
оркестрацию use case’ов.

Для обеспечения взаимодействия с внешними клиентами (web, другие сервисы)
необходимо ввести чёткую HTTP-границу.

Эта граница должна:
- предоставлять стабильный контракт для работы с Trading
- не раскрывать внутренние детали домена и application layer
- не содержать бизнес-логики
- выступать адаптером между HTTP и слоем приложения

## Decision
- Introduce an explicit HTTP adapter layer for the Trading bounded context
- HTTP handlers delegate all business decisions to application use cases
- Handlers are responsible only for:
    - request parsing and validation
    - passing contextual data (headers, params)
    - mapping application/domain errors to HTTP responses
- No domain logic or state transitions are allowed in HTTP handlers
- Security-related headers (e.g. Company ID, User ID) are treated as part of the contract
  but are not validated against external systems at this stage
- HTTP layer depends on the application layer, never on domain internals directly

---

- Ввести явный HTTP-адаптер для bounded context Trading
- HTTP-обработчики полностью делегируют принятие решений слою приложения
- Ответственность HTTP-слоя ограничена:
    - разбором и валидацией входных данных
    - передачей контекстной информации (заголовки, параметры)
    - отображением ошибок домена и application layer в HTTP-ответы
- Бизнес-логика и переходы состояний запрещены в HTTP-обработчиках
- Заголовки безопасности (например, Company ID, User ID) фиксируются как часть контракта,
  но их проверка через внешние системы на данном этапе не выполняется
- HTTP-слой зависит от application layer, но не от внутренних деталей домена

## Consequences
- Trading becomes accessible to external clients via a stable HTTP contract
- Application layer remains the single entry point for all state-changing operations
- Domain model stays fully isolated and testable
- Error handling becomes explicit and standardized at the boundary
- Future migration to BFF, GraphQL, or gRPC is simplified
- Integration with other bounded contexts can rely on the HTTP contract without
  coupling to internal implementations

---

- Trading становится доступным для внешних клиентов через стабильный HTTP-контракт
- Слой приложения остаётся единственной точкой входа для операций изменения состояния
- Доменная модель сохраняет изоляцию и тестируемость
- Обработка ошибок становится явной и стандартизированной на границе
- Упрощается будущий переход к BFF, GraphQL или gRPC
- Интеграция с другими bounded context’ами возможна без зависимости
  от внутренних реализаций Trading
