# Fish Exchange

Монорепозиторий платформы торгов рыбой.  
Архитектура: DDD + transactional outbox + event-driven интеграция между `catalog`, `trading`, `deals`.

## Состав

- `identity` (`:8084`) — компании, пользователи, аутентификация, роли.
- `catalog` (`:8081`) — рыба, продукты, лоты, публикация лотов.
- `trading` (`:8082`) — аукционы, ставки, чтение аукциона по id/лоту.
- `deals` (`:8083`) — сделка после выигрыша аукциона, lifecycle контракта и оплаты.
- `billing` (`:8085`) — кошелёк компании (баланс, тестовый top-up, ledger); JWT как у `catalog`.
- `integration` — relay outbox-событий и фоновые джобы:
  - автозакрытие просроченных аукционов;
  - отмена просроченных сделок и fallback на следующего кандидата;
  - создание счёта в `billing` по событию `identity.CompanyCreated` из outbox.
- `frontend` (`:3000`) — Next.js UI, проксирует запросы в backend.

## Быстрый старт

Требования:

- Docker + Docker Compose
- Go 1.22+ (для локального `go test`)

Поднять весь стек:

```bash
docker compose up -d --build
```

Остановить и удалить volume БД:

```bash
docker compose down -v
```

Запустить тесты:

```bash
go test ./...
```

## Команды Makefile

- `make test` — `go test ./...`
- `make compose-up` — запуск compose с build
- `make compose-down` — остановка compose и удаление volume
- `make demo-happy` — позитивный сценарий
- `make demo-fallback` — fallback на следующего победителя
- `make demo-auto` — автозакрытие аукциона
- `make e2e-bid-race` — race + extension для ставок

## Ключевые бизнес-правила

### Trading

- Аукцион создается асинхронно после публикации лота.
- Первая ставка: `amount >= start_price`.
- Следующая ставка: `amount >= current_price + min_bid_step`.
- В конкурентных ставках используется блокировка строки (`FOR UPDATE`).
- Если ставка пришла в `extension_window`, `ends_at` переносится на `now + extension_duration`.
- После `ends_at` ставка отклоняется.

### Deals

- `AuctionWon` запускает создание `Deal`.
- Контракт готовится отдельной командой `PrepareContract`.
- Дедлайны:
  - `contract_sign_deadline`
  - `payment_deadline`
- При просрочке сделка отменяется фоном, затем запускается fallback на следующего кандидата.
- Fallback срабатывает при любой отмене сделки, если есть активная цепочка кандидатов.

### Identity

- Саморегистрация с ролью `admin` запрещена.
- Админ может повысить пользователя до `admin`.
- Регистрация пользователя без реквизитов компании допустима (пустой `company_id`).
- При старте `identity` создается bootstrap-админ (можно отключить env-переменной).

## Основные HTTP endpoints

### Identity

- `POST /companies`
- `POST /users`
- `POST /auth/login`
- `GET /users/me`
- `GET /users` (только admin)
- `POST /users/{id}/promote-admin` (только admin)

### Catalog

- `POST /fish`
- `POST /products`
- `POST /products/{id}/publish`
- `POST /lots`
- `POST /lots/{id}/publish`

### Trading

- `GET /auctions`
- `POST /auctions/{id}/publish`
- `POST /auctions/{id}/bids`
- `POST /auctions/{id}/close`
- `POST /auctions/{id}/cancel`
- `GET /auctions/{id}`
- `GET /auctions/by-lot/{lotId}`

### Billing

- `GET /billing/accounts/me` — баланс (`available`, `held`, `total`).
- `POST /billing/accounts/me/top-up/test` — тело `{"amount": <int64>}` (тестовое зачисление).
- `GET /billing/accounts/me/ledger` — последние записи ledger (до 100).

### Deals

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

## Переменные окружения (минимум)

Общее для backend:

- `PGHOST`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGPORT`, `PGSSLMODE`
- `IDENTITY_TOKEN_SECRET`
- `IDENTITY_TOKEN_TTL_MINUTES`

Billing:

- `BILLING_PORT` (по умолчанию в compose: `8085`)

Integration:

- `AUCTION_CLOSE_INTERVAL_SEC`
- `AUCTION_CLOSE_BATCH`
- `DEAL_DEADLINE_INTERVAL_SEC`
- `DEAL_DEADLINE_BATCH`

Identity bootstrap admin:

- `IDENTITY_BOOTSTRAP_ADMIN_ENABLED`
- `IDENTITY_BOOTSTRAP_ADMIN_LOGIN`
- `IDENTITY_BOOTSTRAP_ADMIN_PASSWORD`

Frontend:

- `NEXT_PUBLIC_CATALOG_API_URL`
- `NEXT_PUBLIC_TRADING_API_URL`
- `NEXT_PUBLIC_DEALS_API_URL`
- `NEXT_PUBLIC_IDENTITY_API_URL`
- `NEXT_PUBLIC_BILLING_API_URL` (опционально, для следующего этапа UI)

Детали по frontend: `apps/frontend/README.md`.
