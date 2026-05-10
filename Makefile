SHELL := /bin/bash

.PHONY: test backend-test frontend-typecheck compose-up compose-up-chain compose-down chain-setup chain-down demo-happy demo-fallback demo-auto demo-full-payment demo-all e2e-bid-race

test:
	$(MAKE) backend-test
	$(MAKE) frontend-typecheck

backend-test:
	go test ./...

frontend-typecheck:
	test -d apps/frontend/node_modules || npm --prefix apps/frontend ci
	npm --prefix apps/frontend run typecheck

compose-up:
	docker compose up --build

chain-setup:
	./scripts/chain_setup_local.sh

compose-up-chain: chain-setup
	docker compose --env-file .chain/chain.env up -d --build

chain-down:
	docker compose stop anvil || true

compose-down:
	docker compose down -v

demo-happy:
	./scripts/demo_happy_path.sh

demo-fallback:
	./scripts/demo_fallback_winner.sh

demo-auto:
	./scripts/demo_auto_close.sh

demo-full-payment:
	./scripts/demo_full_payment_flow.sh

demo-all: demo-happy demo-fallback demo-auto

e2e-bid-race:
	./scripts/e2e_bid_race_extension.sh
