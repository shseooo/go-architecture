# go-architecture

A **modular monolith** shop API in Go, built with the standard library
(`net/http`, `database/sql`) plus [sqlc](https://sqlc.dev) for type-safe queries
and [goose](https://github.com/pressly/goose) for migrations. It demonstrates
bounded-context modules with enforced boundaries, transactional cross-module
use-cases, dynamic queries, a standardized response envelope, and end-to-end
tests against a real database.

## Bounded contexts

The application is split into self-contained modules; each owns its tables and
exposes a small public API. Other modules depend only on that public API — never
on a module's internals (the Go compiler enforces this via the nested
`internal/` directory).

```
              cmd/api (composition root — wires modules + cross-context adapters)
                 │
     ┌───────────┼────────────┐
     ▼           ▼            ▼
  ordering    customer     catalog     ← contexts; ordering talks to the others
     │  (via consumer-defined gateways, adapted at the root)
     └───────────┬────────────┘
                 ▼
     shared (Address, error taxonomy)  +  platform (database, httpx)
```

Inside each module the layering is the same: `domain` (entities) ← `app`
(use-cases, consumer-defined repository interfaces) ← `repo` (sqlc) / `rest`
(net/http). Cross-context calls (e.g. ordering reserving stock in catalog) run in
one transaction because all modules share a single `*sql.DB` and a context-based
`TxManager`.

## Structure

```
cmd/
  api/main.go        HTTP server (graceful shutdown)
  import/            one-off CSV → DB batch importer
  migrate/           goose migration runner
migrations/          goose SQL migrations (embedded)
sqlc.yaml            sqlc config (schema = migrations, one package per module)
internal/
  platform/
    database/        connection + TxManager (context-based transactions)
    httpx/           response envelope, error mapping, middleware
  shared/            shared kernel: Address value object, error taxonomy
  bootstrap/         wires modules + cross-context adapters into one handler
  catalog/           item / category context   (catalog.go = public API)
    internal/{domain,app,repo,rest}
  customer/          member context            (customer.go = public API)
    internal/{domain,app,repo,rest}
  ordering/          order / delivery context  (ordering.go = public API)
    internal/{domain,repo,rest}
deploy/              Dockerfile, docker-compose.yaml
docs/                generated OpenAPI 2.0 spec (swaggo)
e2e/                 testcontainers end-to-end tests (build tag: e2e)
```

## API

| Area   | Endpoint |
|--------|----------|
| Member | `POST /members`, `GET /members/{id}` |
| Item   | `POST /items`, `PUT /items/{id}`, `GET /items/{id}`, `GET /items?categoryId=&minPrice=&maxPrice=&sort=&limit=&offset=` |
| Order  | `POST /orders`, `GET /members/{id}/orders`, `POST /orders/{id}/cancel` |
| Misc   | `GET /healthz`, `GET /swagger/index.html` |

Responses use a consistent envelope:

```jsonc
{ "data": { … } }                                   // single
{ "data": [ … ] }                                   // array
{ "data": [ … ], "meta": { "pagination": { … } } }  // paged
{ "error": { "code": "NOT_FOUND", "message": "…" } }// error
```

## Getting started

Requires Go 1.26+, Docker.

```bash
git clone https://github.com/shseooo/go-architecture.git
cd go-architecture
cp example.env .env

make up          # start MySQL (docker-compose) + hot-reloading app (air)
make migrate     # apply goose migrations
```

```bash
curl localhost:9090/items
open http://localhost:9090/swagger/index.html
```

## Development

```bash
make tests        # unit tests (services, fast, no external deps)
make test-e2e     # end-to-end tests against a real MySQL (requires Docker)
make migrate      # goose up          (migrate-status for status)
make swagger      # regenerate docs/  from handler annotations
sqlc generate     # regenerate the typed DB layer from queries + migrations
go run ./cmd/import -file cmd/import/items.sample.csv   # CSV import
```

## Tech stack

- **HTTP**: standard library `net/http` (Go 1.22+ routing), no framework
- **DB**: `database/sql` + `go-sql-driver/mysql`, queries via **sqlc**
- **Migrations**: **goose** (embedded, shared by the app and tests)
- **Docs**: `swaggo/swag` + `swaggo/http-swagger`
- **Tests**: `testify`, `testcontainers-go`

---

Based on the Clean Architecture idea popularized by
[bxcodec/go-clean-arch](https://github.com/bxcodec/go-clean-arch); reworked into
a modular-monolith shop domain on the standard library + sqlc/goose.
