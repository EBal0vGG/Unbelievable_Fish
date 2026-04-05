SHELL := /bin/bash

.PHONY: test compose-up compose-down demo-happy demo-fallback demo-auto demo-all

test:
	go test ./...

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down -v

demo-happy:
	./scripts/demo_happy_path.sh

demo-fallback:
	./scripts/demo_fallback_winner.sh

demo-auto:
	./scripts/demo_auto_close.sh

demo-all: demo-happy demo-fallback demo-auto
