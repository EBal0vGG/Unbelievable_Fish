# Trading Application Layer

The application layer defines the boundary between the outside world
(API, messaging, CLI, jobs) and the trading domain model.

It is responsible for orchestrating use cases, loading and saving aggregates,
and publishing domain events — without containing any business logic itself.

---
Слой приложения определяет границу между внешним миром
(API, сообщения, CLI, фоновые задачи) и торговой доменной моделью.

Он отвечает за оркестрацию сценариев использования, загрузку и сохранение агрегатов,
а также публикацию доменных событий — не содержа при этом бизнес-логики.

---

## Architecture Principles

- **Domain logic lives only in the domain layer**
- **Application layer contains no business rules**
- **Aggregates are modified only through explicit use cases**
- **Side effects are triggered by domain events**
- **Dependencies point inward (Infrastructure → Application → Domain)**

---

- **Вся бизнес-логика находится только в доменном слое**
- **Слой приложения не содержит бизнес-правил**
- **Агрегаты изменяются только через явные use cases**
- **Побочные эффекты инициируются доменными событиями**
- **Зависимости направлены внутрь (Инфраструктура → Приложение → Домен)**

---

## Use Cases

Each use case represents a single application-level action
and follows the same execution flow:

1. Load aggregate from repository
2. Invoke a domain method
3. Persist aggregate state
4. Publish returned domain events

---

Каждый use case представляет одно прикладное действие
и следует одинаковому потоку выполнения:

1. Загрузка агрегата из репозитория
2. Вызов доменного метода
3. Сохранение состояния агрегата
4. Публикация возвращённых доменных событий

### Available Use Cases

- `CreateAuction`
- `PublishAuction`
- `PlaceBid`
- `CloseAuction`
- `CancelAuction`

Each use case exposes a single `Execute(...)` method
and contains no branching business logic.

---


Каждый use case предоставляет один метод `Execute(...)`
и не содержит бизнес-логики.

---

## Repositories

Repositories are defined as **application-owned interfaces**.
They abstract persistence concerns and allow the domain to remain
pure and infrastructure-agnostic.

```go
type AuctionRepository interface {
    Load(ctx context.Context, id AuctionID) (*auction.Auction, error)
    Save(ctx context.Context, a *auction.Auction) error
}
```

The domain has no knowledge of repositories or storage mechanisms.

---


Репозитории определены как **интерфейсы слоя приложения**.
Они абстрагируют работу с хранилищем и позволяют домену
оставаться чистым и независимым от инфраструктуры.

Домен ничего не знает о репозиториях и способах хранения данных.

---

## Event Publishing

The application layer publishes domain events returned by aggregates
via the `EventPublisher` interface.

```go
type EventPublisher interface {
    Publish(ctx context.Context, events []auction.Event) error
}
```

The publisher may later be implemented as:

* in-memory dispatcher
* message broker (Kafka, NATS, RabbitMQ)
* outbox pattern
* synchronous event handlers

The application layer does not depend on any specific implementation.

---


Слой приложения публикует доменные события,
возвращаемые агрегатами, через интерфейс `EventPublisher`.

Реализация publisher может быть любой:

* in-memory диспетчер
* брокер сообщений (Kafka, NATS, RabbitMQ)
* outbox-паттерн
* синхронные обработчики событий

Слой приложения не зависит от конкретной реализации.

---

## Extensibility

The system is **closed for modification and open for extension**.

* New auction types can be added without changing existing ones
* New aggregates introduce new use cases, not changes to old ones
* The application layer is agnostic to specific auction implementations
* Domain events enable evolutionary and event-driven architectures

---

Система **закрыта для модификаций и открыта для расширений**.

* Новые типы торгов добавляются без изменения существующих
* Новые агрегаты приводят к новым use cases, а не к правкам старых
* Слой приложения не зависит от конкретных реализаций аукционов
* Доменные события позволяют эволюционное и событийное развитие системы


