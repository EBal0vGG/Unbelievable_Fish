# ADR-0002: Domain Events as First-Class Citizens

## Context
The trading domain is implemented as a pure Go domain model and represents
the core business logic of the platform.

As the system evolves, it is necessary to clearly separate:
- decision-making based on business rules
- execution of side effects (persistence, messaging, integration)

The domain must remain isolated from infrastructure concerns and must not
orchestrate actions outside of its own consistency boundaries.

---

Торговая доменная область реализована как чистая доменная модель Go и является
ядром бизнес-логики платформы.

По мере развития системы требуется чёткое разделение:
- принятия решений на основе бизнес-правил
- выполнения побочных эффектов (персистентность, сообщения, интеграции)

Домен должен оставаться изолированным от инфраструктурных деталей и не должен
оркестрировать действия за пределами своей зоны согласованности.

## Decision
- Domain events are treated as first-class citizens of the domain model
- Aggregate methods return domain events to signal business facts
- Domain events represent facts that have already happened, not commands or intentions
- Domain events contain data only and no business logic
- Aggregates do not invoke application services, infrastructure, or side effects
- Application layer reacts to domain events and performs required actions
- Persistence and messaging are handled outside the domain as adapters

---

- Доменные события рассматриваются как first-class citizens (объекты первого типа) доменной модели
- Методы агрегатов возвращают доменные события, фиксирующие бизнес-факты
- Доменные события описывают произошедшие факты, а не команды или намерения
- Доменные события содержат только данные и не включают бизнес-логику
- Агрегаты не вызывают сервисы приложения, инфраструктуру или побочные эффекты
- Слой приложения реагирует на доменные события и выполняет необходимые действия
- Персистентность и обмен сообщениями реализуются вне домена в виде адаптеров

## Consequences
- Aggregate method signatures explicitly communicate business outcomes
- A dedicated application layer becomes required to orchestrate use cases
- Business logic remains fully testable without infrastructure
- Domain model becomes suitable for event-driven and asynchronous architectures
- Transition to event sourcing or outbox patterns becomes easier
- Domain events become part of the ubiquitous language of the system

---

- Сигнатуры методов агрегатов явно отражают бизнес-результаты
- Появляется обязательный слой приложения для оркестрации сценариев
- Бизнес-логика остаётся полностью тестируемой без инфраструктуры
- Доменная модель готова к событийной и асинхронной архитектуре
- Упрощается переход к event sourcing или outbox-паттернам
- Доменные события становятся частью ubiquitous language системы

