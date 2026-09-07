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

.PHONY: prepare
prepare: prepare
	@echo "Checking httpyac..."
	@if command -v httpyac >/dev/null 2>&1; then \
		echo "httpyac is already installed"; \
	else \
		echo "Installing httpyac..."; \
		npm install httpyac -g; \
	fi

	@echo "Checking golang-migrate..."
	@if command -v migrate >/dev/null 2>&1; then \
		echo "golang-migrate is already installed"; \
	elif [ "$$(uname -s)" = "Darwin" ]; then \
		echo "macOS detected. Installing golang-migrate with Homebrew..."; \
		if ! command -v brew >/dev/null 2>&1; then \
			echo "Error: Homebrew is not installed."; \
			exit 1; \
		fi; \
		brew install golang-migrate; \
	else \
		echo "Installing golang-migrate with go install..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi

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

.PHONY: test_rest
test_rest:
	cd ./etc &&	httpyac ./user-svc.http --env dev

.PHONY: fmt
fmt:
	go fmt ./...

