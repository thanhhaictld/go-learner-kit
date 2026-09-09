# User Service

A production-oriented Go microservice template using:

- Go
- Gin
- OpenAPI-first development
- `oapi-codegen`
- PostgreSQL
- GORM
- OpenTelemetry
- OTLP / OpenTelemetry Collector
- `log/slog`
- SQL migrations
- Docker Compose

The API contract is defined first in `api/openapi.yaml`. Go HTTP types and Gin route bindings are generated from that contract.

---

## Architecture

```text
                    OpenAPI
                 api/openapi.yaml
                       |
                       | go generate
                       v
                api/generated
                       |
                       v
                    Gin
                       |
                OTel Middleware
                       |
                       v
                   Handler
                       |
                       v
                   Service
                       |
                       v
             Repository Interface
                       |
                       v
               GORM Repository
                       |
                       v
                  PostgreSQL
```

Dependency direction:

```text
HTTP -> Service -> Repository abstraction
                      ^
                      |
                PostgreSQL/GORM
```

Keep transport, business logic, and persistence concerns separated.

---

## Project Structure

```text
user-service/
├── cmd/
│   └── api/
│       └── main.go
│
├── api/
│   ├── openapi.yaml
│   ├── oapi-codegen.yaml
│   ├── generate.go
│   └── generated/
│       └── api.gen.go
│
├── internal/
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   └── postgres.go
│   │
│   ├── domain/
│   │   └── user.go
│   │
│   ├── repository/
│   │   ├── user_repository.go
│   │   └── postgres/
│   │       └── user_repository.go
│   │
│   ├── service/
│   │   └── user_service.go
│   │
│   ├── telemetry/
│   │   └── otel.go
│   │
│   └── transport/
│       └── http/
│           └── handler.go
│
├── migrations/
│   ├── 000001_create_users.up.sql
│   └── 000001_create_users.down.sql
│
├── deploy/
│   └── otel-collector.yaml
│
├── compose.yaml
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

---

# Prerequisites

Install:

- Go 1.25+
- Docker / Docker Compose
- PostgreSQL client tools, optional
- `golang-migrate`, if migrations are executed from the host

Verify:

```bash
go version
docker version
docker compose version
```

---

# First-Time Setup

Clone the repository:

```bash
git clone <repository-url>
cd user-service
```

Download dependencies:

```bash
go mod download
```

Verify dependencies:

```bash
go mod tidy
```

Generate OpenAPI code:

```bash
go generate ./...
```

Start infrastructure:

```bash
docker compose up -d postgres otel-collector
```

Run database migrations:

```bash
make migrate-up
```

Start the application:

```bash
make run
```

Start it in watch mode (rebuilds and restarts after Go source changes):

```bash
make watch
```

Or:

```bash
go run ./cmd/api
```

The API should be available at:

```text
http://localhost:8080
```

## Organization authorization

Run the complete local stack from the repository root:

```bash
cp .env.example .env
docker compose up --build
```

The initial administrator and organization are configured through the two UUIDs in `.env`. Requests to user endpoints require the trusted `X-User-ID` and `X-Organization-ID` headers. The development authorization service is private to the Compose network.

The organization migration assumes development data is disposable. If a database already contains users from before this change, reset its development volume before starting the stack:

```bash
docker compose down -v
```

---

# Configuration

Development configuration is supplied through environment variables.

Example:

```bash
export DB_HOST=localhost
export DB_PORT=5453
export DB_USER=app
export DB_PASSWORD=app
export DB_NAME=users
export DB_SSLMODE=disable


export OTEL_SERVICE_NAME=user-service
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_INSECURE=true
```

Typical production configuration should come from Kubernetes ConfigMaps and Secrets rather than committed files.

Do not commit credentials.

---

# Development Workflow

The normal development workflow is:

```text
Change API contract
       |
       v
api/openapi.yaml
       |
       v
go generate ./...
       |
       v
Implement/fix generated interface
       |
       v
Service / domain logic
       |
       v
Repository / persistence
       |
       v
Tests
       |
       v
go fmt / go vet
       |
       v
Run locally
```

---

## 1. Change the API Contract

API changes start in:

```text
api/openapi.yaml
```

Example:

```yaml
paths:
  /users/{id}:
    get:
      operationId: getUser
```

Always define a stable `operationId`.

Good:

```yaml
operationId: getUser
```

Avoid unnamed operations because generated method names become harder to control.

---

## 2. Generate Go Code

The generator configuration lives in:

```text
api/oapi-codegen.yaml
```

Recommended configuration:

```yaml
package: generated

generate:
  models: true
  gin-server: true
  strict-server: true

output: api/generated/api.gen.go
```

Generation is declared through:

```text
api/generate.go
```

```go
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
```

Generate code:

```bash
go generate ./...
```

Generated files must not be edited manually.

```text
api/generated/*
```

should always be treated as generated source.

If the OpenAPI contract changes, regenerate the code instead of modifying generated structs or interfaces.

---

## 3. Implement the Generated Handler Contract

The generated server interface is implemented under:

```text
internal/transport/http/
```

Handlers are responsible for:

- HTTP request/response mapping
- request validation
- converting API DTOs to application/domain values
- mapping application errors to HTTP responses

Handlers should not contain business rules.

Example flow:

```text
HTTP request
    |
    v
Generated OpenAPI request
    |
    v
Handler
    |
    v
Service
```

Keep handlers thin.

---

## 4. Add Business Logic

Business logic belongs under:

```text
internal/service/
```

Example:

```go
func (s *UserService) Create(
    ctx context.Context,
    email string,
    name string,
) (*domain.User, error) {
    // business logic
}
```

The service layer should not depend on:

- Gin
- HTTP status codes
- GORM models
- OpenAPI-generated response models

Services depend on repository abstractions.

---

## 5. Update the Domain Model

Domain objects live under:

```text
internal/domain/
```

Example:

```go
type User struct {
    ID        uuid.UUID
    Email     string
    Name      string
    CreatedAt time.Time
}
```

Do not reuse OpenAPI-generated models as domain entities.

Keep these concepts separate:

```text
OpenAPI DTO
    |
    | map
    v
Domain Model
    |
    | map
    v
Database Model
```

This prevents database or HTTP contract changes from leaking through the entire application.

---

# Database Development

PostgreSQL access goes through repository implementations.

```text
internal/repository/
```

Repository abstraction:

```text
internal/repository/user_repository.go
```

PostgreSQL implementation:

```text
internal/repository/postgres/user_repository.go
```

Always pass `context.Context` into database operations:

```go
r.db.
    WithContext(ctx).
    First(&model, "id = ?", id)
```

This preserves:

- request cancellation
- deadlines
- trace propagation

---

# Database Migrations

Do not use GORM `AutoMigrate` as the production schema-management mechanism.

Database changes must use versioned migration files.

Example:

```text
migrations/
├── 000001_create_users.up.sql
└── 000001_create_users.down.sql
```

Create table:

```sql
CREATE TABLE users
(
    id UUID PRIMARY KEY,
    email VARCHAR(320) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX ux_users_email
ON users(email);
```

Rollback:

```sql
DROP TABLE users;
```

Run migrations:

```bash
make migrate-up
```

Rollback one migration:

```bash
make migrate-down
```

Check migration version:

```bash
make migrate-version
```

Recommended workflow for schema changes:

```text
Create migration
      |
      v
Apply locally
      |
      v
Update GORM persistence model
      |
      v
Run tests
      |
      v
Validate rollback
```

Never edit an already-released migration.

Create a new migration instead.

---

# OpenTelemetry

Incoming Gin requests are instrumented through OpenTelemetry middleware.

```go
router.Use(
    otelgin.Middleware("user-service"),
)
```

Telemetry is exported over OTLP:

```text
user-service
     |
     | OTLP
     v
OpenTelemetry Collector
     |
     +----> tracing backend
     |
     +----> metrics backend
     |
     +----> observability vendor
```

The service should not depend directly on Grafana, Jaeger, Tempo, or another observability vendor.

Configure the destination through:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

---

## Context Propagation

Always propagate the same context through application layers:

```text
Gin request
    |
    v
c.Request.Context()
    |
    v
Service
    |
    v
Repository
    |
    v
GORM
    |
    v
PostgreSQL
```

Example:

```go
ctx := c.Request.Context()

user, err := h.users.Get(ctx, id)
```

Repository:

```go
r.db.WithContext(ctx)
```

This allows a distributed trace to remain correlated through the entire request.

---

## Manual Spans

Do not create spans for every small function.

Create spans around meaningful business operations.

Example:

```go
var tracer = otel.Tracer("user-service/service")

func (s *UserService) Get(
    ctx context.Context,
    id uuid.UUID,
) (*domain.User, error) {

    ctx, span := tracer.Start(
        ctx,
        "UserService.Get",
    )

    defer span.End()

    // ...
}
```

Expected trace:

```text
GET /api/v1/users/{id}
|
+-- UserService.Get
    |
    +-- PostgreSQL query
```

---

# Logging

Use Go's standard structured logger:

```text
log/slog
```

Example:

```go
slog.Info(
    "user created",
    "user_id", user.ID,
)
```

Prefer structured attributes over formatted text.

Good:

```go
slog.Error(
    "failed to create user",
    "user_id", user.ID,
    "error", err,
)
```

Avoid:

```go
log.Printf("failed to create user %s: %v", user.ID, err)
```

Useful log fields include:

```text
service
environment
trace_id
span_id
request_id
tenant_id
user_id
```

Never log passwords, access tokens, secrets, or sensitive payloads.

---

# Health Checks

Infrastructure endpoints:

```text
GET /health/live
GET /health/ready
```

Liveness answers:

> Is the process alive?

It should generally not check external dependencies.

Readiness answers:

> Can this process currently serve traffic?

Readiness may verify PostgreSQL connectivity.

Example Kubernetes configuration:

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
```

---

# Running Locally

Start dependencies:

```bash
docker compose up -d postgres otel-collector
```

Run migrations:

```bash
make migrate-up
```

Run service:

```bash
make run
```

Check health:

```bash
curl http://localhost:8080/health/live
```

```bash
curl http://localhost:8080/health/ready
```

Create a user:

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "name": "Alice"
  }'
```

---

# Docker Compose

Run everything:

```bash
docker compose up --build
```

Stop:

```bash
docker compose down
```

Remove containers and local volumes:

```bash
docker compose down -v
```

Use the last command carefully because it removes PostgreSQL development data.

---

# Testing

Run all tests:

```bash
go test ./...
```

Run with race detection:

```bash
go test -race ./...
```

Run a package:

```bash
go test ./internal/service/...
```

Run one test:

```bash
go test ./internal/service/... -run TestUserService_Create
```

Recommended test distribution:

```text
internal/service/
    unit tests

internal/transport/http/
    HTTP handler tests

internal/repository/postgres/
    integration tests

api/
    OpenAPI contract validation
```

Most business rules should be testable without PostgreSQL.

Example:

```text
UserService
    |
    v
Mock/Fake UserRepository
```

This is one reason the repository interface exists.

---

# Formatting

Format all Go code:

```bash
go fmt ./...
```

Optional stricter formatting:

```bash
gofmt -w .
```

---

# Static Analysis

Run:

```bash
go vet ./...
```

Recommended CI also uses `golangci-lint`.

Example:

```bash
golangci-lint run
```

---

# Dependency Management

Add a dependency:

```bash
go get <module>
```

Clean dependency declarations:

```bash
go mod tidy
```

Check available updates:

```bash
go list -m -u all
```

Do not manually edit package versions unless required.

---

# Recommended Makefile

Example:

```makefile
APP=user-service

.PHONY: generate
generate:
	go generate ./...

.PHONY: run
run:
	go run ./cmd/api

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: check
check: generate fmt vet test

.PHONY: compose-up
compose-up:
	docker compose up -d

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: migrate-up
migrate-up:
	migrate \
		-path migrations \
		-database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" \
		up

.PHONY: migrate-down
migrate-down:
	migrate \
		-path migrations \
		-database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" \
		down 1

.PHONY: migrate-version
migrate-version:
	migrate \
		-path migrations \
		-database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" \
		version
```

Normal developer commands then become:

```bash
make generate
make run
make test
make check
```

---

# Typical Feature Workflow

Suppose we need:

```text
GET /users/{id}
```

The development workflow should be:

### 1. Update OpenAPI

```text
api/openapi.yaml
```

### 2. Regenerate

```bash
make generate
```

### 3. Fix compile errors

The generated interface tells you what needs implementing.

### 4. Update handler

```text
internal/transport/http/handler.go
```

### 5. Add service behavior

```text
internal/service/user_service.go
```

### 6. Update repository if needed

```text
internal/repository/
```

### 7. Add database migration if the schema changed

```text
migrations/
```

### 8. Add tests

```bash
make test
```

### 9. Run static checks

```bash
make check
```

### 10. Run locally

```bash
make compose-up
make migrate-up
make run
```

---

# Typical API-Change Workflow

For an API-only contract change:

```text
openapi.yaml
    |
    v
make generate
    |
    v
compiler errors
    |
    v
update handler
    |
    v
tests
```

Do not manually synchronize route definitions and documentation.

OpenAPI is the single source of truth.

---

# Typical Database-Change Workflow

For a database schema change:

```text
New migration
    |
    v
Apply locally
    |
    v
Update GORM persistence model
    |
    v
Update repository
    |
    v
Tests
    |
    v
Validate rollback
```

Do not modify production schemas manually.

---

# Before Opening a Pull Request

Run:

```bash
make generate
make fmt
make vet
make test-race
```

Or:

```bash
make check
```

Also verify:

- generated code is current
- OpenAPI spec is valid
- new migrations have rollback scripts
- no credentials were committed
- tests cover new business behavior
- errors are mapped consistently
- logs do not expose secrets
- request context is propagated
- telemetry still exports correctly

---

# CI Pipeline

A basic CI flow should run:

```text
Checkout
   |
   v
Setup Go
   |
   v
go mod download
   |
   v
go generate ./...
   |
   v
git diff --exit-code
   |
   v
go fmt validation
   |
   v
go vet ./...
   |
   v
go test -race ./...
   |
   v
Build
```

The important check is:

```bash
go generate ./...
git diff --exit-code
```

This fails CI when someone changes `openapi.yaml` but forgets to commit regenerated Go code.

---

# Build

Build locally:

```bash
go build -o bin/user-service ./cmd/api
```

Run:

```bash
./bin/user-service
```

For Linux:

```bash
CGO_ENABLED=0 GOOS=linux go build \
  -o bin/user-service \
  ./cmd/api
```

---

# Graceful Shutdown

The HTTP server should handle:

```text
SIGTERM
   |
   v
Stop accepting requests
   |
   v
Complete active requests
   |
   v
Close/flush telemetry
   |
   v
Exit
```

This behavior is required for reliable Kubernetes rolling deployments.

Do not immediately terminate the process when receiving SIGTERM.

---

# Coding Rules

## Handler

Responsible for HTTP concerns.

```text
HTTP DTO -> Service -> HTTP DTO
```

Do not put business logic here.

## Service

Responsible for application/business behavior.

```text
Service -> Repository interface
```

Do not import Gin or GORM.

## Repository

Responsible for persistence.

```text
Domain model <-> DB model
```

Do not expose GORM objects to the service layer.

## Domain

Contains business concepts and domain values.

Do not depend on:

- Gin
- GORM
- OpenAPI-generated code

---

# Important Rules

### OpenAPI is the API source of truth

Do not manually create duplicate Gin route definitions when generated routes already exist.

### Generated code is read-only

Never edit:

```text
api/generated/
```

### Always propagate context

Use:

```go
ctx := c.Request.Context()
```

and pass it through all application layers.

### Do not use AutoMigrate in production

Use explicit SQL migrations.

### Keep handlers thin

HTTP mapping belongs in handlers.

Business decisions belong in services.

### Keep GORM behind repositories

Do not query the database directly from handlers or services.

### Telemetry should be vendor-neutral

Export through OTLP to an OpenTelemetry Collector.

---

# Quick Command Reference

```bash
# dependencies
go mod download
go mod tidy

# generate OpenAPI code
make generate

# run
make run

# format
make fmt

# static checks
make vet

# tests
make test

# race tests
make test-race

# complete local checks
make check

# infrastructure
make compose-up
make compose-down

# database
make migrate-up
make migrate-down
make migrate-version

# build
go build ./cmd/api
```

---

# Development Philosophy

The service follows four main principles:

```text
1. Contract first
   OpenAPI defines the external API.

2. Compiler assisted
   Generated interfaces make API drift visible during compilation.

3. Dependency separation
   HTTP, business logic, and persistence stay independent.

4. Context everywhere
   Cancellation, deadlines, and distributed tracing flow through every layer.
```

For most microservices, prefer this straightforward layering:

```text
transport
    |
    v
service
    |
    v
repository
```

Only introduce additional architectural layers when domain complexity actually requires them.
