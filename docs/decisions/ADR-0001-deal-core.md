## ADR-0001: Deal as Supporting Domain

**Context**  
Deal (фиксация сделок после аукциона) является поддерживающим доменом платформы. Отвечает за lifecycle сделки после определения победителя.

**Decision**  
Deal domain implemented as pure Go domain model with:
- No framework dependencies
- Immutable product snapshots
- Event-driven construction
- Explicit state machine for deal lifecycle
- API and persistence as adapters

Домен сделок реализован как чистая доменная модель Go:
- Без зависимостей от фреймворков
- Неизменяемые снапшоты продукта
- Конструкторы на основе событий
- Явный конечный автомат жизненного цикла
- API и персистентность как адаптеры

**Consequences**  
- Deal logic isolated from infrastructure
- Immutable product data ensures consistency
- Event-driven integration with Catalog and Trading
- Clear audit trail via status changes

- Логика сделок изолирована от инфраструктуры
- Неизменяемые данные продукта гарантируют консистентность
- Event-driven интеграция с Catalog и Trading
- Четкий аудит через изменения статусов

---
