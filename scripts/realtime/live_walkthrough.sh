#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$ROOT_DIR/common.sh"
require python3
require docker

AUTO="${AUTO:-0}"
CLOSE_MODE="${CLOSE_MODE:-admin}" # admin | auto

FISH_NAME="${FISH_NAME:-Cod}"
FISH_DESCRIPTION="${FISH_DESCRIPTION:-Live demo fish}"
PRODUCT_WEIGHT="${PRODUCT_WEIGHT:-10}"
PRODUCT_UNIT="${PRODUCT_UNIT:-kg}"
PRODUCT_SIZE="${PRODUCT_SIZE:-M}"
PRODUCT_PROCESSING="${PRODUCT_PROCESSING:-frozen}"
SELLER_COMPANY_ID="${SELLER_COMPANY_ID:-seller-1}"
START_PRICE="${START_PRICE:-100}"
QUANTITY="${QUANTITY:-10}"
if [[ -n "${DURATION_MIN+x}" ]]; then
  DURATION_MIN_WAS_SET=1
else
  DURATION_MIN_WAS_SET=0
fi
DURATION_MIN="${DURATION_MIN:-5}"
BID1_AMOUNT="${BID1_AMOUNT:-120}"
BID2_AMOUNT="${BID2_AMOUNT:-150}"
BID1_COMPANY="${BID1_COMPANY:-buyer-1}"
BID1_USER="${BID1_USER:-user-1}"
BID2_COMPANY="${BID2_COMPANY:-buyer-2}"
BID2_USER="${BID2_USER:-user-2}"
DEAL_COMPANY_ID="${DEAL_COMPANY_ID:-seller-1}"
DEAL_USER_ID="${DEAL_USER_ID:-manager-1}"
AUCTION_WAIT_SEC="${AUCTION_WAIT_SEC:-20}"
AUCTION_POLL_INTERVAL_SEC="${AUCTION_POLL_INTERVAL_SEC:-2}"
AUTO_CLOSE_WAIT_SEC="${AUTO_CLOSE_WAIT_SEC:-40}"
AUTO_CLOSE_POLL_INTERVAL_SEC="${AUTO_CLOSE_POLL_INTERVAL_SEC:-2}"
AUTO_CLOSE_GRACE_SEC="${AUTO_CLOSE_GRACE_SEC:-20}"
PLACE_BIDS_IN_AUTO="${PLACE_BIDS_IN_AUTO:-0}"
PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PG_CONTAINER="${PG_CONTAINER:-}"
AUCTION_STARTS_AT_INPUT="${AUCTION_STARTS_AT:-}"

pause() {
  local text="$1"
  if [[ "$AUTO" == "1" ]]; then
    echo
    echo "== $text =="
    return
  fi
  echo
  read -r -p "== $text == (Enter для продолжения) " _
}

json_get() {
  python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get(sys.argv[2], ""))' "$1" "$2"
}

extract_json_field() {
  python3 -c 'import json,sys,re; txt=sys.argv[1]; key=sys.argv[2]; m=re.search(r"\{.*?\}", txt, re.S); print(json.loads(m.group(0)).get(key,"") if m else "")' "$1" "$2"
}

show_hint() {
  echo "Команда: $1"
}

call_json() {
  local cmd="$1"
  show_hint "$cmd"
  eval "$cmd"
}

resolve_pg_container() {
  if [[ -n "$PG_CONTAINER" ]]; then
    return
  fi
  PG_CONTAINER="$(docker compose ps -q postgres)"
  if [[ -z "$PG_CONTAINER" ]]; then
    PG_CONTAINER="fish-postgres-1"
  fi
}

lookup_auction_id_by_lot() {
  local lot="$1"
  docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c \
    "select auction_id from trading_auctions where lot_id = '$lot' order by starts_at desc limit 1;" \
    | tr -d '[:space:]'
}

lookup_auction_state() {
  local auction="$1"
  docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c \
    "select state from trading_auctions where auction_id = '$auction' limit 1;" \
    | tr -d '[:space:]'
}

lookup_auction_ends_at_epoch() {
  local auction="$1"
  docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c \
    "select extract(epoch from ends_at)::bigint from trading_auctions where auction_id = '$auction' limit 1;" \
    | tr -d '[:space:]'
}

pause "Шаг 1/10: создаём fish в Catalog"
fish_resp="$(call_json "$ROOT_DIR/catalog_create_fish.sh \"$FISH_NAME\" \"$FISH_DESCRIPTION\"")"
echo "$fish_resp"
fish_id="$(extract_json_field "$fish_resp" "fish_id")"
if [[ -z "$fish_id" ]]; then
  echo "Не удалось извлечь fish_id" >&2
  exit 1
fi

pause "Шаг 2/10: создаём product в Catalog"
product_resp="$(call_json "$ROOT_DIR/catalog_create_product.sh \"$fish_id\" \"$PRODUCT_WEIGHT\" \"$PRODUCT_UNIT\" \"$PRODUCT_SIZE\" \"$PRODUCT_PROCESSING\"")"
echo "$product_resp"
product_id="$(extract_json_field "$product_resp" "product_id")"
if [[ -z "$product_id" ]]; then
  echo "Не удалось извлечь product_id" >&2
  exit 1
fi

pause "Шаг 3/10: публикуем product"
show_hint "$ROOT_DIR/catalog_publish_product.sh \"$product_id\""
"$ROOT_DIR/catalog_publish_product.sh" "$product_id"

pause "Шаг 4/10: создаём lot"
auction_starts_at_for_lot="$AUCTION_STARTS_AT_INPUT"
duration_min_for_lot="$DURATION_MIN"
if [[ -z "$auction_starts_at_for_lot" && "$CLOSE_MODE" == "auto" ]]; then
  if [[ "$PLACE_BIDS_IN_AUTO" == "1" ]]; then
    # Keep a short active window for bids, then let scheduler close automatically.
    auction_starts_at_for_lot="$(date -u -d '-30 sec' +%Y-%m-%dT%H:%M:%SZ)"
    if [[ "$DURATION_MIN_WAS_SET" == "0" ]]; then
      duration_min_for_lot="1"
    fi
  else
    # In pure auto-close demo mode, make auction already expired.
    auction_starts_at_for_lot="$(date -u -d '-10 min' +%Y-%m-%dT%H:%M:%SZ)"
  fi
fi
lot_resp="$(AUCTION_STARTS_AT="$auction_starts_at_for_lot" call_json "$ROOT_DIR/catalog_create_lot.sh \"$product_id\" \"$SELLER_COMPANY_ID\" \"$START_PRICE\" \"$QUANTITY\" \"$duration_min_for_lot\"")"
echo "$lot_resp"
lot_id="$(extract_json_field "$lot_resp" "lot_id")"
if [[ -z "$lot_id" ]]; then
  echo "Не удалось извлечь lot_id" >&2
  exit 1
fi
if [[ -n "$auction_starts_at_for_lot" ]]; then
  echo "auction_starts_at=$auction_starts_at_for_lot"
fi
echo "auction_duration_minutes=$duration_min_for_lot"

pause "Шаг 5/10: публикуем lot (запускается цепочка событий)"
show_hint "$ROOT_DIR/catalog_publish_lot.sh \"$lot_id\""
"$ROOT_DIR/catalog_publish_lot.sh" "$lot_id"

echo
echo "Авто-поиск auction_id по lot_id в Postgres..."
resolve_pg_container
auction_id=""
attempts=$(( (AUCTION_WAIT_SEC + AUCTION_POLL_INTERVAL_SEC - 1) / AUCTION_POLL_INTERVAL_SEC ))
for _ in $(seq 1 "$attempts"); do
  auction_id="$(lookup_auction_id_by_lot "$lot_id" || true)"
  if [[ -n "$auction_id" ]]; then
    break
  fi
  sleep "$AUCTION_POLL_INTERVAL_SEC"
done

if [[ -z "$auction_id" ]]; then
  echo "Не удалось авто-извлечь auction_id за ${AUCTION_WAIT_SEC}s."
  echo "Введите auction_id вручную (или проверьте integration/postgres)."
  read -r -p "auction_id: " auction_id
fi
if [[ -z "${auction_id:-}" ]]; then
  echo "auction_id обязателен" >&2
  exit 1
fi
echo "auction_id=$auction_id"

if [[ "$CLOSE_MODE" != "admin" && "$CLOSE_MODE" != "auto" ]]; then
  echo "Неверный CLOSE_MODE=$CLOSE_MODE (допустимо: admin|auto)" >&2
  exit 2
fi

if [[ "$CLOSE_MODE" == "admin" || "$PLACE_BIDS_IN_AUTO" == "1" ]]; then
  pause "Шаг 6: делаем ставку #1"
  show_hint "$ROOT_DIR/trading_place_bid.sh \"$auction_id\" \"$BID1_AMOUNT\" \"$BID1_COMPANY\" \"$BID1_USER\""
  "$ROOT_DIR/trading_place_bid.sh" "$auction_id" "$BID1_AMOUNT" "$BID1_COMPANY" "$BID1_USER"

  pause "Шаг 7: делаем ставку #2"
  show_hint "$ROOT_DIR/trading_place_bid.sh \"$auction_id\" \"$BID2_AMOUNT\" \"$BID2_COMPANY\" \"$BID2_USER\""
  "$ROOT_DIR/trading_place_bid.sh" "$auction_id" "$BID2_AMOUNT" "$BID2_COMPANY" "$BID2_USER"
fi

if [[ "$CLOSE_MODE" == "admin" ]]; then
  pause "Шаг 8: закрываем аукцион через admin CLI"
  show_hint "$ROOT_DIR/admin_close_auction.sh \"$auction_id\""
  "$ROOT_DIR/admin_close_auction.sh" "$auction_id"
else
  pause "Шаг 8: ждём автоматического закрытия scheduler'ом"
  now_epoch="$(date -u +%s)"
  ends_at_epoch="$(lookup_auction_ends_at_epoch "$auction_id" || true)"
  wait_budget="$AUTO_CLOSE_WAIT_SEC"
  if [[ -n "$ends_at_epoch" && "$ends_at_epoch" =~ ^[0-9]+$ ]]; then
    remaining=$(( ends_at_epoch - now_epoch + AUTO_CLOSE_GRACE_SEC ))
    if (( remaining > wait_budget )); then
      wait_budget="$remaining"
    fi
  fi
  echo "Ожидание state != PUBLISHED (таймаут ${wait_budget}s, grace=${AUTO_CLOSE_GRACE_SEC}s)..."
  closed_state=""
  close_attempts=$(( (wait_budget + AUTO_CLOSE_POLL_INTERVAL_SEC - 1) / AUTO_CLOSE_POLL_INTERVAL_SEC ))
  for _ in $(seq 1 "$close_attempts"); do
    closed_state="$(lookup_auction_state "$auction_id" || true)"
    if [[ -n "$closed_state" && "$closed_state" != "PUBLISHED" ]]; then
      break
    fi
    sleep "$AUTO_CLOSE_POLL_INTERVAL_SEC"
  done
  if [[ -z "$closed_state" || "$closed_state" == "PUBLISHED" ]]; then
    echo "Автозакрытие не произошло за ${wait_budget}s" >&2
    echo "Подсказка: проверьте, что integration запущен и auction уже истёк по ends_at" >&2
    exit 1
  fi
  echo "Аукцион закрыт автоматически, state=$closed_state"
fi

pause "Шаг 9: смотрим projection в Deals"
show_hint "$ROOT_DIR/deals_get_projection.sh \"$auction_id\""
"$ROOT_DIR/deals_get_projection.sh" "$auction_id"

pause "Шаг 10: смотрим сделку по auction_id"
deal_resp="$("$ROOT_DIR/deals_get_by_auction.sh" "$auction_id")"
echo "$deal_resp"

deal_id="$(python3 -c 'import json,sys,re; txt=sys.argv[1]; m=re.search(r"\{.*\}", txt, re.S); print((json.loads(m.group(0)).get("deal_id","") if m else ""))' "$deal_resp" 2>/dev/null || true)"
if [[ -n "$deal_id" ]]; then
  echo "deal_id=$deal_id"
  echo
  echo "Опционально подтвердить сделку:"
  echo "$ROOT_DIR/deals_confirm.sh \"$deal_id\" \"$DEAL_COMPANY_ID\" \"$DEAL_USER_ID\""
else
  echo "deal_id не извлечён автоматически (проверьте ответ выше)."
fi

echo
echo "Готово. ID этого прогона:"
echo "fish_id=$fish_id"
echo "product_id=$product_id"
echo "lot_id=$lot_id"
echo "auction_id=$auction_id"
