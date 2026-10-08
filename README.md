# gRPC Duo

gRPC Duo is a small, production-minded Go backend project that demonstrates how a public REST API can call an internal gRPC service backed by PostgreSQL.

The domain is intentionally simple: product catalog data lives in `catalog-service`, pricing data lives in `pricing-service`, and the two services communicate through a versioned Protocol Buffers contract. The value of the project is in the engineering around that flow: context propagation, deadlines, error mapping, health checks, readiness, Docker, tests, observability, CI, and security-conscious defaults.

## What This Project Demonstrates

- REST as the public API boundary.
- gRPC and Protocol Buffers for internal service-to-service communication.
- A small multi-service Go workspace with separate service modules.
- PostgreSQL-backed repository code with parameterized SQL.
- Context propagation and request deadlines across HTTP, gRPC, and database calls.
- Deterministic gRPC-to-HTTP error mapping.
- Health and readiness checks for service orchestration.
- Structured logs with request ID propagation.
- Unit and integration tests for core behavior and failure paths.
- Docker Compose local execution with hardened application containers.
- GitHub Actions CI for formatting, tests, vet, vulnerability checks, builds, and Docker images.

## Architecture

```text
HTTP Client
    |
    | GET /products/{id}
    v
catalog-service
    | REST API
    | product lookup in memory
    |
    | pricing.v1.PricingService/GetPrice
    v
pricing-service
    | gRPC API
    | PostgreSQL repository
    v
PostgreSQL
```

Services:

- `catalog-service`: public HTTP service. It owns the REST API and in-memory product metadata.
- `pricing-service`: internal gRPC service. It owns price lookup and PostgreSQL access.
- `postgres`: local PostgreSQL database seeded for demo requests.

## Why REST Externally And gRPC Internally

REST is used at the external boundary because it is easy to call from browsers, scripts, API clients.

gRPC is used internally because it gives the service-to-service call a typed, versioned, generated contract. In this project, that contract makes the dependency between catalog and pricing explicit without exposing the internal pricing service directly to the public API.

## Request Flow

For a successful request:

1. The client calls `GET /products/42`.
2. `catalog-service` validates the product ID.
3. `catalog-service` loads product metadata from its in-memory catalog.
4. `catalog-service` calls `pricing-service` through the generated gRPC client.
5. `pricing-service` validates the product ID.
6. `pricing-service` queries PostgreSQL with a parameterized query.
7. `pricing-service` returns price data over gRPC.
8. `catalog-service` combines product and price data into one JSON response.

Request IDs flow through the same path:

```text
X-Request-ID header
  -> catalog-service HTTP context
  -> x-request-id gRPC metadata
  -> pricing-service context
  -> PostgreSQL query logs
```

## Running Locally

Requirements:

- Go with the configured toolchain available.
- Docker Desktop or a compatible Docker engine.
- Docker Compose.

Start the full stack:

```bash
docker compose up --build
```

The public API is available at:

```text
http://localhost:8080
```

Call the main flow:

```bash
curl -H "X-Request-ID: demo-123" http://localhost:8080/products/42
```

Stop the stack:

```bash
docker compose down
```

Reset local PostgreSQL data:

```bash
docker compose down -v
```

Docker Compose publishes HTTP and PostgreSQL only on `127.0.0.1`. The `pricing-service` gRPC port is not published to the host; it is reachable only on the internal Docker network.

## API Examples

Successful product lookup:

```bash
curl -i -H "X-Request-ID: demo-123" http://localhost:8080/products/42
```

Expected response body:

```json
{
  "id": "42",
  "name": "Wireless Mouse",
  "description": "Ergonomic wireless mouse",
  "price": {
    "amount_cents": 12990,
    "currency": "BRL",
    "discount_percent": 10
  }
}
```

Unknown product:

```bash
curl -i http://localhost:8080/products/999
```

Expected status:

```text
HTTP/1.1 404 Not Found
```

Expected body:

```json
{
  "error": {
    "code": "product_not_found",
    "message": "product not found"
  }
}
```

Invalid product ID:

```bash
curl -i http://localhost:8080/products/bad%20id
```

Expected status:

```text
HTTP/1.1 400 Bad Request
```

Health and readiness:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

`/health` confirms that `catalog-service` is alive. `/ready` confirms that `catalog-service` can reach `pricing-service`, whose own gRPC health status depends on PostgreSQL readiness.

## Protobuf Contract

The pricing contract lives at:

```text
proto/pricing/v1/pricing.proto
```

It defines:

```protobuf
service PricingService {
  rpc GetPrice(GetPriceRequest) returns (GetPriceResponse);
}

message GetPriceRequest {
  string product_id = 1;
}

message GetPriceResponse {
  string product_id = 1;
  int64 amount_cents = 2;
  string currency = 3;
  int32 discount_percent = 4;
}
```

Generated Go code is committed under:

```text
gen/go/pricing/v1
```

Regenerate the Protobuf code:

```bash
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
buf generate
```

## Testing

Run the fast test suites:

```bash
cd services/catalog-service
go test ./...
```

```bash
cd services/pricing-service
go test ./...
```

The default tests cover:

- product and price validation;
- happy paths;
- product and price not found behavior;
- repository errors;
- gRPC status code mapping;
- gRPC client deadlines;
- HTTP error mapping;
- response JSON shape;
- request ID propagation;
- health and readiness behavior.

`catalog-service` includes an in-memory integration test using `bufconn`. It exercises the real HTTP handler, generated gRPC client, and an in-process gRPC server without opening a network port.

Run the optional PostgreSQL integration test:

```bash
docker compose up -d postgres
```

```powershell
cd services/pricing-service
$env:PRICING_INTEGRATION_DATABASE_URL="postgres://grpc_duo_app:grpc_duo_dev_password@localhost:5432/grpc_duo?sslmode=disable"
go test -tags=integration ./internal/postgres
```

On non-PowerShell shells, set `PRICING_INTEGRATION_DATABASE_URL` with the syntax used by your shell. The integration test runs real migrations before testing repository behavior.

## Quality And CI

Run the local quality script:

```powershell
./scripts/quality.ps1
```

Apply Go formatting:

```powershell
./scripts/quality.ps1 -Fix
```

Include Docker image builds:

```powershell
./scripts/quality.ps1 -Docker
```

Include vulnerability checks:

```powershell
./scripts/quality.ps1 -Security
```

The GitHub Actions workflow at `.github/workflows/ci.yml` runs on pushes to `main` and on pull requests. It checks:

- `gofmt`;
- `go mod verify`;
- `go test ./...`;
- `go vet ./...`;
- `govulncheck`;
- `go build ./cmd/...`;
- Docker builds for both service images.
