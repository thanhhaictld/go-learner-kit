APP_NAME=user-service


MIGRATIONS_DIR=migrations
OPENAPI_CONFIG=api/oapi-codegen.yaml
OPENAPI_SPEC=api/openapi.yaml

DB_HOST ?= localhost
DB_PORT ?= 5453
DB_USER ?= app
DB_PASSWORD ?= app
DB_NAME ?= users
DB_SSLMODE ?= disable

DATABASE_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# Make variables are not automatically inherited by recipe processes. Export the
# development defaults so the API receives the same connection settings as the
# migration targets.
export DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE

.PHONY: generate
generate: generate-openapi


.PHONY: generate-openapi
generate-openapi:
	go tool oapi-codegen \
		-config $(OPENAPI_CONFIG) \
		$(OPENAPI_SPEC)

.PHONY: migration
migration:
ifndef name
	$(error name is required. Example: make migration name=create_users)
endif
	migrate create \
		-ext sql \
		-dir $(MIGRATIONS_DIR) \
		-seq \
		$(name)

.PHONY: migrate-up
migrate-up:
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		up

.PHONY: migrate-down
migrate-down:
	migrate \
		-path $(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		down 1

.PHONY: run
run:
	go run ./cmd/api

.PHONY: watch
watch:
	go tool air -c .air.toml

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	go fmt ./...
