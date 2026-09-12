# Go Learner Kit

A small SaaS microservice workspace for learning API design, tenant isolation, OpenID Connect authentication, and relationship-based authorization. It contains a Go user API, an ASP.NET Core/OpenIddict identity service, and a private OpenFGA authorization service.

## Architecture

```mermaid
flowchart LR
    Client[Client or API gateway]
    Identity[identity-service<br/>ASP.NET Identity + OpenIddict]
    IdentityDB[(Identity PostgreSQL)]
    User[user-service<br/>Gin + OpenAPI]
    UserDB[(User PostgreSQL)]
    Authz[authz-service<br/>Gin + OpenAPI]
    FGA[OpenFGA]
    FGADB[(OpenFGA PostgreSQL)]
    Prometheus[Prometheus]
    Grafana[Grafana]

    Client -->|OIDC login / token| Identity
    Identity -->|accounts, organizations,<br/>memberships, OAuth data| IdentityDB
    Client -->|trusted identity headers<br/>(current user API)| User
    User -->|create, list, get| UserDB
    User -->|permission check| Authz
    Authz -->|check and write tuples| FGA
    FGA --> FGADB
    User -->|/metrics| Prometheus
    Identity -->|/metrics| Prometheus
    Authz -->|/metrics| Prometheus
    FGA -->|/metrics| Prometheus
    UserDB -->|postgres exporter| Prometheus
    IdentityDB -->|postgres exporter| Prometheus
    FGADB -->|postgres exporter| Prometheus
    Prometheus --> Grafana
```

`user-service` is exposed at `http://localhost:8080` and `identity-service` at `http://localhost:8081`. `authz-service`, OpenFGA, and all databases stay on the Compose network.

`identity-service` owns accounts, credentials, organizations, memberships, and OpenID Connect tokens. The current `user-service` routes still require trusted identity headers injected by a gateway or another trusted caller; bearer-token validation is the next integration step:

```http
X-User-ID: <user UUID>
X-Organization-ID: <organization UUID>
```

OpenFGA stores role assignments as organization relationships. In the initial model, an organization `admin` can `list_users` and `create_user`; retrieving a user requires `list_users` and is restricted to the requested organization.

## Custom roles

Organization administrators can create a UUID-backed custom role, grant it `list_users` and/or `create_user`, then assign it to users. Role management stays with direct organization admins, so a custom role cannot grant itself more permissions.

```text
POST /v1/organizations/{organizationId}/roles
PUT  /v1/organizations/{organizationId}/roles/{roleId}/permissions/{permission}
PUT  /v1/organizations/{organizationId}/roles/{roleId}/users/{userId}
```

These private `authz-service` routes require the acting administrator in `X-User-ID`. See [authz-svc/README.md](authz-svc/README.md) for the full API, including revocation routes.

## Services

| Service | Responsibility |
| --- | --- |
| `identity-svc` | ASP.NET Core Identity and OpenIddict provider. It provides the Razor account UI, organization selection, OIDC discovery/JWKS, and authorization-code + PKCE tokens. |
| `user-svc` | Creates, lists, and retrieves organization-scoped users. It fails closed if authorization is denied or unavailable. |
| `authz-svc` | Initializes the OpenFGA model, seeds an initial admin, checks permissions, and provides private admin role assignment endpoints, including identity-service organization-owner provisioning. |
| OpenFGA | Evaluates organization relationship tuples. |

## Run locally

Prerequisites: Docker and Docker Compose.

```bash
cp .env.example .env
docker compose up --build
```

The user API is available at `http://localhost:8080`. Identity UI is available at [http://localhost:8081/Identity/Account/Login](http://localhost:8081/Identity/Account/Login); OIDC discovery is at [http://localhost:8081/.well-known/openid-configuration](http://localhost:8081/.well-known/openid-configuration). The default bootstrap values in `.env.example` identify the first organization and its administrator. Replace the example password and internal provisioning token before sharing a deployment.

The stack runs OpenFGA's one-shot schema migration before starting OpenFGA. If you started a previous version of the stack and saw `relation "store" does not exist`, apply this update and restart the stack; the `openfga-migrate` service creates the missing tables.

Create a user with the seeded administrator:

```bash
curl -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: 11111111-1111-1111-1111-111111111111' \
  -H 'X-Organization-ID: 22222222-2222-2222-2222-222222222222' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com"}'
```

## Development notes

The user migration that adds organization ownership assumes a disposable development database. Reset Compose volumes before applying it to a database created by an earlier version:

```bash
docker compose down -v
docker compose up --build
```

Run service tests independently:

```bash
(cd src/user-svc && go test ./...)
(cd src/authz-svc && go test ./...)
dotnet test src/identity-svc/Identity.Svc.csproj
```

See [user-svc/README.md](src/user-svc/README.md), [authz-svc/README.md](src/authz-svc/README.md), and [identity-svc/README.md](src/identity-svc/README.md) for service-specific details.

## Load testing

The [k6/user-service.js](k6/user-service.js) scenario verifies the ready probe, then repeatedly creates, retrieves, and lists users with the configured organization administrator. It generates a unique email for every iteration.

After starting the Compose stack, run:

```bash
k6 run k6/user-service.js
```

Override the target and load profile as needed:

```bash
BASE_URL=http://localhost:8080 \
USER_ID=11111111-1111-1111-1111-111111111111 \
ORGANIZATION_ID=22222222-2222-2222-2222-222222222222 \
VUS=25 HOLD=2m k6 run k6/user-service.js
```

The default thresholds require fewer than 1% failed HTTP requests, 95th-percentile request latency below 500 ms, and more than 99% successful checks. k6 exposes these configuration values through `__ENV` and exits non-zero when a threshold fails. [k6 environment variables](https://grafana.com/docs/k6/latest/using-k6/environment-variables/), [k6 thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/)

The root `Makefile` manages the whole stack and runs k6 from Docker. After `make up`, send exactly one authenticated `GET /users` request with:

```bash
make k6-once
```

## Monitoring

`make up` starts Prometheus, Grafana, PostgreSQL exporters, and service metrics alongside the application stack. Grafana is available at [http://localhost:3000](http://localhost:3000) with `admin` / `admin`; anonymous viewing is also enabled for local development. Prometheus is available at [http://localhost:9090](http://localhost:9090).

The provisioned **Go Learner Kit - Services and k6** dashboard shows scrape health for Prometheus, all services, OpenFGA, and their databases; Go service request rate, p95 latency, and 5xx errors; and k6 request rate and latency.

The Makefile writes k6 metrics directly to Prometheus whenever it runs a scenario. Run a one-request smoke test or the full workflow, then select the dashboard time range that covers the run:

```bash
make k6-once
make k6
make k6-100rps
```

`make k6-100rps` runs one `GET /users` request per iteration at a constant arrival rate of 100 requests per second for 20 seconds. Override the defaults with `LOAD_RATE` and `LOAD_DURATION` when needed.

For a monitoring-only startup after the application stack is already running, use `make monitor`. k6 uses its Prometheus remote-write output and Prometheus enables its remote-write receiver for local development. [k6 Prometheus remote write](https://grafana.com/docs/k6/latest/results-output/real-time/prometheus-remote-write/), [OpenFGA metrics configuration](https://openfga.dev/docs/getting-started/setup-openfga/configuration)
