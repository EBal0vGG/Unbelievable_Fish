# ADR-0001: Trading as Core Domain

## Context
Trading (auctions, bids, winner selection) is the core business domain
of the platform.
---
Торговля (аукционы, ставки, выбор победителя) является основной бизнес-областью
платформы.

## Decision
- Trading domain implemented as pure Go domain model
- No framework dependencies
- Explicit state machine
- Domain events emitted by aggregates
- API and persistence are adapters
---
- Торговая область реализована как чистая доменная модель Go
- Отсутствие зависимостей от фреймворков
- Явный конечный автомат
- События домена генерируются агрегатами
- API и персистентность являются адаптерами

## Consequences
- OpenAPI defined after domain model
- Trading logic isolated from infrastructure
---
- OpenAPI определен после доменной модели
- Логика торговли изолирована от инфраструктуры
