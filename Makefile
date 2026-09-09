COMPOSE ?= docker compose
K6_IMAGE ?= grafana/k6:latest
K6_NETWORK ?= go-learner-kit_default

USER_ID ?= 11111111-1111-1111-1111-111111111111
ORGANIZATION_ID ?= 22222222-2222-2222-2222-222222222222

.PHONY: help up down restart build ps logs generate fmt test test-user test-authz k6 k6-once

help:
	@echo "make up        Start the full development stack"
	@echo "make down      Stop the development stack"
	@echo "make build     Build service images"
	@echo "make logs      Follow all service logs"
	@echo "make generate  Generate both OpenAPI bindings"
	@echo "make fmt       Format both Go services"
	@echo "make test      Run both Go test suites"
	@echo "make k6        Run the full k6 user workflow"
	@echo "make k6-once   Send one authenticated GET /users request with k6"

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
	$(MAKE) -C user-svc generate
	$(MAKE) -C authz-svc generate

fmt:
	$(MAKE) -C user-svc fmt
	$(MAKE) -C authz-svc fmt

test: test-user test-authz

test-user:
	$(MAKE) -C user-svc test

test-authz:
	$(MAKE) -C authz-svc test

k6:
	docker run --rm --network $(K6_NETWORK) \
		-e BASE_URL=http://user-service:8080 \
		-e USER_ID=$(USER_ID) \
		-e ORGANIZATION_ID=$(ORGANIZATION_ID) \
		-v "$(CURDIR)/k6:/scripts:ro" \
		$(K6_IMAGE) run /scripts/user-service.js

k6-once:
	docker run --rm --network $(K6_NETWORK) \
		-e BASE_URL=http://user-service:8080 \
		-e USER_ID=$(USER_ID) \
		-e ORGANIZATION_ID=$(ORGANIZATION_ID) \
		-v "$(CURDIR)/k6:/scripts:ro" \
		$(K6_IMAGE) run --vus 1 --iterations 1 /scripts/smoke.js
