SHELL := /bin/bash

.PHONY: client-install typecheck test build db-up db-down server-run client-dev

client-install:
	cd client && npm install

typecheck:
	cd client && npm run typecheck
	cd server && go test ./... >/dev/null

test:
	cd client && npm run test
	cd server && go test ./...

build:
	cd client && npm run build
	cd server && go build ./...

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

server-run:
	cd server && go run ./cmd/spanpay-api

client-dev:
	cd client && npm run dev
