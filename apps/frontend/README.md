# Frontend MVP

Next.js frontend для MVP рыбной биржи. UI собран как thin client над существующими сервисами `catalog`, `trading`, `deals` и не дублирует доменную логику во фронте.

## Что реализовано

- Упрощенный `login/register` через локальный user context: `companyId` + `userId`
- Header/navigation с отображением текущего контекста
- Главная marketplace-страница
- Разделы `Каталог`, `Лоты`, `Аукционы`
- Формы:
  - `Создать рыбу`
  - `Создать продукт` как helper внутри `Создать лот`
  - `Создать лот`
  - `Создать аукцион`
  - `Сделать ставку`
- Страница деталей аукциона с polling/refetch и bid history
- Страница `Мой контекст / мои действия`
- Graceful fallback на mock/local read model там, где backend query/flow не доведен до UI

## Структура

```text
apps/frontend/src
  app/                 # Next app router + API proxy routes
  shared/              # config, fetch client, ui kit, utils, types
  entities/            # fish / lot / auction / session hooks + cards
  features/            # auth, filters, create forms, place bid
  widgets/             # header, dashboard sections
```

## Как запустить

Требования:

- Node.js 20+
- поднятые backend сервисы из корня репозитория

Пример:

```bash
docker compose up -d postgres migrate catalog trading deals integration
cd apps/frontend
cp .env.example .env.local
npm install
npm run dev
```

Frontend стартует на `http://localhost:3000`.

## Env

См. `.env.example`:

```bash
NEXT_PUBLIC_CATALOG_API_URL=http://localhost:8081
NEXT_PUBLIC_TRADING_API_URL=http://localhost:8082
NEXT_PUBLIC_DEALS_API_URL=http://localhost:8083
NEXT_PUBLIC_ENABLE_API_FALLBACK=true
NEXT_PUBLIC_ENABLE_COMMAND_FALLBACK=false
```

Важно:

- браузер не ходит напрямую в backend, а использует Next proxy routes `/api/catalog/*`, `/api/trading/*`, `/api/deals/*`
- это снижает риск CORS-проблем и оставляет frontend тонким адаптером
- команды записи (create/publish/place bid) по умолчанию **без fallback**: если backend недоступен, UI покажет ошибку, а не локальный mock-успех
- в этом режиме сидовые mock-данные скрываются из списков, чтобы UI не подставлял несуществующие backend ID в команды записи

## Backend matrix

### Реально подключено к backend

`catalog`

- `POST /fish`
- `POST /products`
- `POST /products/{productID}/publish`
- `POST /lots`
- `POST /lots/{lotID}/publish`

`trading`

- `POST /auctions/{id}/bids`

`deals`

- `GET /deal-projections/{auctionId}`
- `GET /deals/by-auction/{auctionId}`

### Есть в OpenAPI/handlers, но сейчас не считаются надежно доступными для UI

`trading`

- `POST /auctions`
- `POST /auctions/{id}/publish`
- `POST /auctions/{id}/close`
- `POST /auctions/{id}/cancel`
- `GET /auctions/{id}`

Причина:

- фактический `cmd/trading/main.go` сейчас wiring'ит только `PlaceBid`
- create flow не возвращает стабильный `auction_id` в HTTP response
- list/read-model endpoint для списка аукционов отсутствует
- в strict command mode frontend не вызывает `POST /auctions` и возвращает явную ошибку с рекомендацией: публиковать лот в Catalog (аукцион создается integration-цепочкой)

### Пока работают через mock / local mirror

- список fish/catalog
- список lots
- список auctions
- manual create auction screen
- bid history list
- recent actions / my context read model

## Где real API, а где mock

`real API first`

- команды создания и публикации в `catalog`
- `place bid` в `trading`
- projection/deal read-side в `deals`

`mock / stub fallback`

- любые list endpoints, которых backend пока не отдает
- аукционы, если нужен список или детальная страница без готового query route
- регистрация пользователя
- сценарии, где backend команда принимается, но не возвращает идентификатор сущности обратно в UI

## Временные заглушки

В коде оставлены явные TODO-комментарии и source notes для мест, где нужен backend:

- `shared/api/catalog-service.ts`
- `shared/api/trading-service.ts`
- `shared/api/mock-store.ts`

UI в этих местах не переносит бизнес-правила на фронт, а только:

- показывает страницу и форму
- делает реальный запрос, если route доступен
- иначе включает безопасный local placeholder

## Ограничения текущего backend для UI

- `catalog` не отдает list/query endpoints для marketplace
- `trading` не отдает стабильный read-model списка аукционов
- `CreateAuction` не возвращает `auction_id`
- real registration/auth отсутствуют

Из-за этого фронт хранит временный session context и локально зеркалит созданные сущности, чтобы MVP был usable до завершения backend query-side.
