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

Тесты `*_RealPG` и часть `internal/billing/postgres` при отсутствии `PGHOST`/`PGUSER`/`PGDATABASE` подключаются к Postgres с **дефолтами как в `docker-compose.yml`**: `127.0.0.1:5433`, пользователь/пароль/БД `fish`. Перед прогоном: `docker compose up -d postgres`. Если база недоступна, такие тесты делают `Skip` с подсказкой, а не падают.

## Команды Makefile

- `make test` — `go test ./...`
- `make compose-up` — запуск compose с build
- `make compose-down` — остановка compose и удаление volume
- `make demo-happy` — позитивный сценарий
- `make demo-full-payment` — полный платёжный поток: инвойс (fake), payout admin ready/paid, баланс продавца
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

- `GET /fish` — справочник (без JWT).
- `POST /fish` — только **admin** (справочник платформы).
- `GET /products`, `GET /lots` — **JWT обязателен**; выдача фильтруется по компании/роли; владение продуктом/лотом проверяется в домене при мутациях.
- `POST /products`, `POST /products/{id}/publish`, `POST /lots`, `POST /lots/{id}/publish` — **seller** (и выше по политике identity) + проверки владения на сервере.

### Trading

- `GET /auctions`, `GET /auctions/{id}`, `GET /auctions/by-lot/{lotId}` — чтение (с JWT).
- `POST /auctions/{id}/publish` — seller.
- `POST /auctions/{id}/bids` — buyer.
- `POST /auctions/{id}/close` — только **admin** (или внутренний **system**-метаданные в домене); продавец не может досрочно закрыть активные торги со ставками.
- `POST /auctions/{id}/cancel` — **seller или admin**; продавец: черновик или опубликованный лот **без ставок**; при ставках — отмена через админа / системные правила.

### Billing

- `GET /billing/accounts/me` — баланс (`available`, `held`, `total`); при fake-провайдере в ответе может быть `top_up_fake_confirm_enabled` / `deal_invoice_fake_confirm_enabled` для UI.
- `POST /billing/top-ups` — создание заявки на пополнение (авторизованная компания); при fake — `POST /billing/top-ups/{id}/fake-confirm`.
- `POST /billing/accounts/me/top-up/test` — **только dev/тест**, не основной UX.
- `GET /billing/accounts/me/ledger` — последние записи ledger (до 100).

### Deals

- `GET /deal-projections/{auctionId}`
- `GET /deals/by-auction/{auctionId}`
- `GET /deals/{dealId}`
- `POST /deals/{dealId}/confirm`
- `POST /deals/{dealId}/contract/prepare`
- `POST /deals/{dealId}/contract/sign`
- `POST /deals/{dealId}/payment/request`
- Переход сделки в оплаченное состояние — **только из биллинга** (событие «invoice paid» → домен deals). Отдельного публичного HTTP `mark-paid` нет.
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
- `NEXT_PUBLIC_BILLING_URL` (база billing API, см. `apps/frontend/.env.example`)
- `NEXT_PUBLIC_ENABLE_API_FALLBACK` — локальный fallback при недоступности API
- `NEXT_PUBLIC_ENABLE_FAKE_BILLING=true` — показывать кнопки fake-confirm только вместе с fake-провайдером на сервере
- `NEXT_PUBLIC_ENABLE_BILLING_ADMIN=true` — секция админ-операций биллинга в UI (нужны admin JWT и `BILLING_ENABLE_ADMIN_ACTIONS`)

Детали по frontend: `apps/frontend/README.md`.
