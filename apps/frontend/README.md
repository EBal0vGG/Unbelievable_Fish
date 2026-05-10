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
- Graceful fallback на `mock-store` только когда API недоступен или нет сессии (см. ниже); после успешного `GET /products` и `GET /lots` списки синхронизируются с сервером

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
NEXT_PUBLIC_BILLING_URL=http://localhost:8085/billing
NEXT_PUBLIC_ENABLE_API_FALLBACK=true
# Опционально: demo / fake billing UI (должно совпадать с fake-провайдером на billing)
NEXT_PUBLIC_ENABLE_FAKE_BILLING=true
# Опционально: админ-панель биллинга (admin JWT + BILLING_ENABLE_ADMIN_ACTIONS)
NEXT_PUBLIC_ENABLE_BILLING_ADMIN=true
```

Важно:

- браузер не ходит напрямую в backend, а использует Next proxy routes `/api/catalog/*`, `/api/trading/*`, `/api/deals/*`, `/api/identity/*`, `/api/billing/*`
- это снижает риск CORS-проблем и оставляет frontend тонким адаптером
- **Без входа** списки продуктов/лотов берутся из локального `mock-store` (демо): серверные `GET /products` и `GET /lots` требуют JWT

## Backend matrix

### Реально подключено к backend

`identity`

- `POST /companies`
- `POST /users`
- `POST /auth/login`
- `GET /users/me`

`catalog`

- `GET /fish` (публично)
- `POST /fish` — только admin
- `GET /products`, `GET /lots` — JWT; данные с сервера — источник истины при залогиненном пользователе
- `POST /products`, `POST /products/{productID}/publish`, `POST /lots`, `POST /lots/{lotID}/publish` — seller + проверки владения на сервере

`trading`

- `GET /auctions`, `GET /auctions/{id}`, `GET /auctions/by-lot/{lotId}`
- `POST /auctions/{id}/publish`, `POST /auctions/{id}/bids`
- `POST /auctions/{id}/close` — admin (домен также разрешает system-закрытие по расписанию)
- `POST /auctions/{id}/cancel` — seller или admin (ограничения в домене)

`deals`

- `GET /deal-projections/{auctionId}`
- `GET /deals/by-auction/{auctionId}`
- `GET /deals/{dealId}`
- команды lifecycle: confirm, contract, sign, `payment/request`, shipment, complete, cancel, price
- оплаченность сделки выставляется **только** из биллинга (нет публичного `payment/mark-paid`)

`billing` (через `/api/billing/*`)

- `GET /accounts/me`, top-ups, invoices (см. `shared/api/billing-service.ts`)
- fake- и admin-действия в UI только при соответствующих env на клиенте и сервере

### Частично через mock / local mirror

- **Гость:** продукты и лоты — из `mock-store`, пока нет JWT (не отражают серверное состояние).
- список аукционов / детали — fallback при ошибках или отключённом API, если включён `NEXT_PUBLIC_ENABLE_API_FALLBACK`
- bid history — локально, если нет отдельного read API
- deals list для компании собирается из известных аукционов + `GET /deals/by-auction`; глобального `GET /deals` нет
- recent actions / my context read model

## Где real API, а где mock

`real API first`

- identity/auth flow целиком
- команды создания и публикации в `catalog`
- `place bid` в `trading`
- projection/deal read-side в `deals`
- command lifecycle сделок в `deals`

`mock / stub fallback`

- сценарии без JWT или при недоступности API (если включён fallback)
- аукционы при сбоях read API
- список сделок — как выше
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

## Ограничения и расхождения с сервером

- `GET /products` / `GET /lots` требуют JWT: **гостевой marketplace показывает демо из `mock-store`**, а не live-данные.
- `trading`: список аукционов и детали зависят от доступности API; при fallback — локальное зеркало.
- `deals` не отдаёт list endpoint для всех сделок компании.
- frontend хранит access token и подставляет Bearer в защищённые запросы; успешные ответы каталога дополнительно пишутся в `mock-store` для офлайн-консистентности форм.

Источник истины для залогиненного пользователя по продуктам/лотам — ответы **catalog** после успешного GET; до входа или при полном отказе API UI может расходиться с backend.
