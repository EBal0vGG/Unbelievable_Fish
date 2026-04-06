# Realtime Endpoint Scripts

Скрипты вызывают основные endpoint'ы по одному, чтобы удобно показывать сценарий вживую на демо.

## Быстрый запуск

Из корня проекта:

```bash
docker compose up -d --build
```

## Основные скрипты

- `scripts/realtime/catalog_create_fish.sh [name] [description]`
- `scripts/realtime/catalog_create_product.sh <fish_id> [weight] [unit] [size] [processing_type]`
- `scripts/realtime/catalog_publish_product.sh <product_id>`
- `scripts/realtime/catalog_create_lot.sh <product_id> [seller_company_id] [start_price] [quantity] [duration_min]`
- `scripts/realtime/catalog_publish_lot.sh <lot_id>`
- `scripts/realtime/trading_place_bid.sh <auction_id> [amount] [company_id] [user_id]`
- `scripts/realtime/deals_get_projection.sh <auction_id>`
- `scripts/realtime/deals_get_by_auction.sh <auction_id>`
- `scripts/realtime/deals_confirm.sh <deal_id> [company_id] [user_id]`
- `scripts/realtime/admin_close_auction.sh <auction_id>`
- `scripts/realtime/live_walkthrough.sh` — интерактивный пошаговый сценарий

## Пример пошаговой демо-цепочки

```bash
./scripts/realtime/catalog_create_fish.sh
./scripts/realtime/catalog_create_product.sh <fish_id>
./scripts/realtime/catalog_publish_product.sh <product_id>
./scripts/realtime/catalog_create_lot.sh <product_id>
./scripts/realtime/catalog_publish_lot.sh <lot_id>
./scripts/realtime/trading_place_bid.sh <auction_id> 120 buyer-1 user-1
./scripts/realtime/admin_close_auction.sh <auction_id>
./scripts/realtime/deals_get_by_auction.sh <auction_id>
```

Интерактивный прогон целиком:

```bash
./scripts/realtime/live_walkthrough.sh
```

Автоматический прогон без пауз:

```bash
AUTO=1 ./scripts/realtime/live_walkthrough.sh
```

Режимы завершения аукциона:

```bash
# По умолчанию: force-close через admin CLI
CLOSE_MODE=admin ./scripts/realtime/live_walkthrough.sh

# Автозавершение через scheduler (integration)
CLOSE_MODE=auto AUTO_CLOSE_WAIT_SEC=60 ./scripts/realtime/live_walkthrough.sh
```

Примечание: в режиме `CLOSE_MODE=auto` скрипт по умолчанию ставит `AUCTION_STARTS_AT` на `-10 min`,
чтобы аукцион был просрочен и scheduler закрыл его быстро. Это можно переопределить своей переменной `AUCTION_STARTS_AT`.
Если включён `PLACE_BIDS_IN_AUTO=1`, скрипт вместо этого создаёт короткое активное окно для ставок
(`start ~ -30 sec`, `duration=1 min`, если `DURATION_MIN` не задан вручную), а затем ждёт автозакрытие.

## Переменные окружения

- `CATALOG_URL` (default `http://localhost:8081`)
- `TRADING_URL` (default `http://localhost:8082`)
- `DEALS_URL` (default `http://localhost:8083`)

Для `admin_close_auction.sh`:

- `PGHOST` (default `localhost`)
- `PGUSER` (default `fish`)
- `PGPASSWORD` (default `fish`)
- `PGDATABASE` (default `fish`)
- `PGPORT` (default `5433`)
- `PGSSLMODE` (default `disable`)

Для `live_walkthrough.sh` (авто-поиск `auction_id`):

- `PGUSER` (default `fish`)
- `PGDATABASE` (default `fish`)
- `PG_CONTAINER` (optional; по умолчанию `docker compose ps -q postgres`, fallback `fish-postgres-1`)
- `AUCTION_WAIT_SEC` (default `20`)
- `AUCTION_POLL_INTERVAL_SEC` (default `2`)
- `CLOSE_MODE` (`admin` по умолчанию, либо `auto`)
- `AUTO_CLOSE_WAIT_SEC` (default `40`)
- `AUTO_CLOSE_POLL_INTERVAL_SEC` (default `2`)
- `AUTO_CLOSE_GRACE_SEC` (default `20`, запас после `ends_at` для scheduler/relay)
- `PLACE_BIDS_IN_AUTO` (default `0`, set `1` чтобы делать ставки и в режиме `auto`)
