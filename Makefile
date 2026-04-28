SHELL := /bin/bash

.PHONY: test compose-up compose-down demo-happy demo-fallback demo-auto demo-all e2e-bid-race

test:
	go test ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down -v

demo-happy:
	./scripts/demo_happy_path.sh

demo-fallback:
	./scripts/demo_fallback_winner.sh

demo-auto:
	./scripts/demo_auto_close.sh

demo-all: demo-happy demo-fallback demo-auto

e2e-bid-race:
	./scripts/e2e_bid_race_extension.sh
