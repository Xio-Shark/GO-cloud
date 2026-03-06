.PHONY: fmt test build up down logs compose-config

fmt:
	gofmt -w cmd internal pkg

test:
	go test ./...

build:
	go build ./cmd/...

up:
	docker compose up -d --build

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=200

compose-config:
	docker compose config
