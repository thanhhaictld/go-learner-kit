COMPOSE ?= docker compose
K6_IMAGE ?= grafana/k6:latest
K6_NETWORK ?= go-saas-kit_default

USER_ID ?= 11111111-1111-1111-1111-111111111111
ORGANIZATION_ID ?= 22222222-2222-2222-2222-222222222222
LOAD_RATE ?= 100
LOAD_DURATION ?= 100s

.PHONY: help up down restart build ps logs generate fmt test test-user test-authz test-identity monitor k6 k6-once k6-100rps

help:
	@echo "make up        Start the full development stack"
	@echo "make down      Stop the development stack"
	@echo "make build     Build service images"
	@echo "make logs      Follow all service logs"
	@echo "make generate  Generate both OpenAPI bindings"
	@echo "make fmt       Format both Go services"
	@echo "make test      Run all service test suites"
	@echo "make monitor   Start Prometheus and Grafana"
	@echo "make k6        Run the full k6 user workflow"
	@echo "make k6-once   Send one authenticated GET /users request with k6"
	@echo "make k6-100rps Send 100 GET /users requests per second for 20 seconds"

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

restart: down up

build:
	$(COMPOSE) build

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

generate:
	$(MAKE) -C src/user-svc generate
	$(MAKE) -C src/authz-svc generate

fmt:
	$(MAKE) -C src/user-svc fmt
	$(MAKE) -C src/authz-svc fmt

test: test-user test-authz test-identity

test-user:
	$(MAKE) -C src/user-svc test

test-authz:
	$(MAKE) -C src/authz-svc test

test-identity:
	dotnet test src/identity-svc/Identity.Svc.csproj

monitor:
	$(COMPOSE) up -d prometheus grafana

k6:
	docker run --rm --network $(K6_NETWORK) \
		-e BASE_URL=http://user-service:8080 \
		-e USER_ID=$(USER_ID) \
		-e ORGANIZATION_ID=$(ORGANIZATION_ID) \
		-e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
		-e 'K6_PROMETHEUS_RW_TREND_STATS=p(95),p(99),min,max' \
		-v "$(CURDIR)/tests/k6:/scripts:ro" \
		$(K6_IMAGE) run -o experimental-prometheus-rw /scripts/user-service.js

k6-once:
	docker run --rm --network $(K6_NETWORK) \
		-e BASE_URL=http://user-service:8080 \
		-e USER_ID=$(USER_ID) \
		-e ORGANIZATION_ID=$(ORGANIZATION_ID) \
		-e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
		-e 'K6_PROMETHEUS_RW_TREND_STATS=p(95),p(99),min,max' \
		-v "$(CURDIR)/tests/k6:/scripts:ro" \
		$(K6_IMAGE) run -o experimental-prometheus-rw --vus 1 --iterations 1 /scripts/smoke.js

k6-100rps:
	docker run --rm --network $(K6_NETWORK) \
		-e BASE_URL=http://user-service:8080 \
		-e USER_ID=$(USER_ID) \
		-e ORGANIZATION_ID=$(ORGANIZATION_ID) \
		-e LOAD_RATE=$(LOAD_RATE) \
		-e LOAD_DURATION=$(LOAD_DURATION) \
		-e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
		-e 'K6_PROMETHEUS_RW_TREND_STATS=p(95),p(99),min,max' \
		-v "$(CURDIR)/tests/k6:/scripts:ro" \
		$(K6_IMAGE) run -o experimental-prometheus-rw /scripts/rate.js
