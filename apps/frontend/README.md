# Frontend

Next.js frontend для B2B marketplace рыбной биржи. UI собран как thin client над сервисами `identity`, `catalog`, `trading`, `deals` и не дублирует доменную логику во фронте.

## Что реализовано

- Реальный `identity/auth` flow через backend: `register company`, `register user`, `login`, `me`
- Header/navigation с отображением текущего контекста
- Главная marketplace-страница с торговой лентой, лотами и продуктами
- Разделы `Каталог`, `Лоты`, `Аукционы`, `Твои сделки`
- Сделки с lifecycle: подтверждение, контракт, подпись, оплата, отгрузка, завершение, отмена, изменение цены
- Список и детали сделок видны только авторизованной компании-участнику: покупателю или поставщику
- Формы:
  - `Создать рыбу`
  - `Создать продукт` как helper внутри `Создать лот`
  - `Создать лот`
  - `Создать аукцион`
  - `Сделать ставку`
- Страница деталей аукциона с polling/refetch и bid history
- Страница деталей сделки с timeline и action panel по Deals-командам
- Страница `Мой контекст / мои действия`
- Graceful fallback на mock/local read model там, где backend query/flow не доведен до UI

## Структура

```text
apps/frontend/src
  app/                 # Next app router + API proxy routes
  shared/              # config, fetch client, ui kit, utils, types
  entities/            # fish / lot / auction / deal / session hooks + cards
  features/            # auth, filters, create forms, place bid, deal actions
  widgets/             # header, dashboard sections
```

## Как запустить

Требования:

- Node.js 20+
- поднятые backend сервисы из корня репозитория

Пример:

```bash
docker compose up -d postgres migrate identity catalog trading deals frontend
```

Frontend стартует на `http://localhost:3000`.

Если нужен только backend без UI:

```bash
docker compose up -d postgres migrate identity catalog trading deals
```

## Env

См. `.env.example` для локального запуска без Docker:

```bash
NEXT_PUBLIC_CATALOG_API_URL=http://localhost:8081
NEXT_PUBLIC_TRADING_API_URL=http://localhost:8082
NEXT_PUBLIC_DEALS_API_URL=http://localhost:8083
NEXT_PUBLIC_IDENTITY_API_URL=http://localhost:8084
NEXT_PUBLIC_ENABLE_API_FALLBACK=true
```

Важно:

- браузер не ходит напрямую в backend, а использует Next proxy routes `/api/catalog/*`, `/api/trading/*`, `/api/deals/*`, `/api/identity/*`
- это снижает риск CORS-проблем и оставляет frontend тонким адаптером

## Backend matrix

### Реально подключено к backend

`identity`

- `POST /companies`
- `POST /users`
- `POST /auth/login`
- `GET /users/me`

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
- `GET /deals/{dealId}`
- `POST /deals/{dealId}/confirm`
- `POST /deals/{dealId}/contract/prepare`
- `POST /deals/{dealId}/contract/sign`
- `POST /deals/{dealId}/payment/request`
- `POST /deals/{dealId}/payment/mark-paid`
- `POST /deals/{dealId}/shipment/request`
- `POST /deals/{dealId}/shipment/mark-shipped`
- `POST /deals/{dealId}/complete`
- `POST /deals/{dealId}/cancel`
- `POST /deals/{dealId}/price`

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

### Пока работают через mock / local mirror

- список fish/catalog
- список lots
- список auctions
- manual create auction screen
- bid history list
- deals list для текущей компании, потому что backend пока не отдает `GET /deals`
- recent actions / my context read model

## Где real API, а где mock

`real API first`

- identity/auth flow целиком
- команды создания и публикации в `catalog`
- `place bid` в `trading`
- projection/deal read-side в `deals`
- command lifecycle сделок в `deals`

`mock / stub fallback`

- любые list endpoints, которых backend пока не отдает
- аукционы, если нужен список или детальная страница без готового query route
- список твоих сделок собирается из известных аукционов через `GET /deals/by-auction/{auctionId}` и локальное зеркало, затем фильтруется по текущей компании
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
- `deals` не отдает list endpoint для всех сделок
- frontend хранит access token и подставляет Bearer token в защищенные запросы

Из-за этого фронт хранит временный session context и локально зеркалит созданные сущности, чтобы marketplace оставался usable до завершения backend query-side.
