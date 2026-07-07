# go-architecture

A clean-architecture sample shop API in Go, built with the **standard library only**
(`net/http`, `database/sql`) — no web framework, no ORM. It demonstrates layered
boundaries, transactional use-cases, dynamic queries, and end-to-end tests against a
real database.

## Layers & data flow

```
                 ┌─────────────────────────────────────────────┐
  HTTP request → │ rest      (net/http handlers, JSON, routing) │
                 │   ↓ depends on interfaces                    │
                 │ service   (business rules, transactions)     │
                 │   ↓ depends on interfaces                    │
                 │ repository/mysql (database/sql)              │
                 └───────────────────┬─────────────────────────┘
                                     ↓
                                 domain  (entities — no dependencies)
```

- Dependencies point **inward**: `domain` imports nothing; outer layers depend on
  interfaces they define themselves (consumer-defined interfaces).
- The web framework and SQL live only at the edges, so they are swappable and every
  layer is unit-testable with a stub.

## Project structure

```
cmd/
  api/main.go        HTTP server entrypoint (graceful shutdown)
  import/            one-off CSV → DB batch importer (+ items.sample.csv)
config/db.go         MySQL connection
bootstrap/           wires repositories → services → handlers into one http.Handler
schema.sql           database schema (member / item / order …)
deploy/              Dockerfile, docker-compose.yaml
docs/                generated OpenAPI 2.0 spec (swaggo)
app/
  domain/            entities: member, item, order, category, delivery
  service/{member,item,order}/   use-cases + repository interfaces
  repository/mysql/  database/sql implementations + TxManager
  rest/              net/http handlers, middleware, router
e2e/                 testcontainers-based end-to-end tests (build tag: e2e)
```

## Domain

A small shop: **members** place **orders** for **items** (books, albums, movies via
single-table inheritance), organized into **categories**, shipped via a **delivery**.

Highlights:
- **Stock management** — stock is decremented on order and restored on cancel.
- **Transactions** — placing/canceling an order commits stock, delivery and order
  rows atomically via a context-based `TxManager`.
- **No N+1** — order listings batch-load their items and deliveries with `IN` queries.
- **Dynamic queries** — item search filters by category/price range and sorts by
  date/price, with an allow-listed `ORDER BY` (injection-safe).

## API

| Area   | Endpoint |
|--------|----------|
| Member | `POST /members`, `GET /members/{id}` |
| Item   | `POST /items`, `PUT /items/{id}`, `GET /items/{id}`, `GET /items?categoryId=&minPrice=&maxPrice=&sort=` |
| Order  | `POST /orders`, `GET /members/{id}/orders`, `POST /orders/{id}/cancel` |
| Misc   | `GET /healthz`, `GET /swagger/index.html` |

## Getting started

Requires Go 1.24+ (uses `net/http` method routing) and Docker.

```bash
# clone
git clone https://github.com/shseooo/go-architecture.git
cd go-architecture

# configure
cp example.env .env

# run MySQL + hot-reloading app via docker-compose
make up
```

The server listens on `:9090`. Try:

```bash
curl localhost:9090/items
open http://localhost:9090/swagger/index.html
```

## Testing

```bash
make tests        # unit tests (services, fast, no external deps)
make test-e2e     # end-to-end tests against a real MySQL (requires Docker)
```

Unit tests drive the services through stubbed repositories. The E2E suite spins up a
real MySQL with [testcontainers](https://golang.testcontainers.org/), calls the HTTP
API, and asserts **both the response and the actual database state**.

## CSV importer

A single-run batch job under `cmd/import` loads items from a CSV through the item
service (same validation as the API):

```bash
go run ./cmd/import -file cmd/import/items.sample.csv
```

## API docs (Swagger)

Annotations live on the handlers; regenerate the OpenAPI 2.0 spec with:

```bash
make swagger      # runs swag init → docs/
```

## Tech stack

- **HTTP**: standard library `net/http` (Go 1.22+ routing)
- **DB**: standard library `database/sql` + `go-sql-driver/mysql`
- **Docs**: `swaggo/swag` + `swaggo/http-swagger`
- **Tests**: `testify`, `testcontainers-go`

---

Based on the Clean Architecture idea popularized by
[bxcodec/go-clean-arch](https://github.com/bxcodec/go-clean-arch); reworked to a shop
domain on the standard library.
