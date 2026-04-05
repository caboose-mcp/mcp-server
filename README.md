# mcp-server

An MCP (Model Context Protocol) server written in Go, plus a companion REST API backed by PostgreSQL. Ships two tools — `calculate` and `dad_joke` — available over both the MCP stdio transport (for Claude Desktop) and HTTP (for JMeter load testing or any HTTP client).

MCP is a JSON-RPC 2.0 protocol that lets AI clients like Claude call typed tools in an external process. The stdio transport is deliberate: the client spawns your server as a child process and pipes JSON over stdin/stdout. No ports, no auth, no network exposure.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ | Build and run both services |
| mise | any | Pins `golangci-lint`, `lefthook`, and Go versions from `mise.toml` |
| Docker + Compose | any | Runs Postgres and the API container |
| swag | v1.16+ | Regenerates Swagger docs after annotation changes |

Install mise from [mise.jdx.dev](https://mise.jdx.dev/getting-started.html).

---

## Setup

```sh
cp .env.example .env        # fill in real credentials (see Security section)
mise install                # installs golangci-lint + lefthook at pinned versions
go mod tidy                 # fetches dependencies into the module cache
lefthook install            # wires git hooks from lefthook.yml
```

**Git hooks (`lefthook.yml`):**
- `pre-commit` — runs `go fmt`, `go vet`, `golangci-lint` in parallel; blocks commit on failure
- `pre-push` — runs `go test ./...`

---

## Project structure

```
mcp-server/
├── cmd/
│   ├── api/
│   │   ├── main.go          # REST API entry point — chi router, graceful shutdown, Swagger mount
│   │   └── docs/            # Generated Swagger files (docs.go, swagger.json, swagger.yaml)
│   │                        # Do not edit by hand — regenerate with: mise run swagger
│   └── mcp/
│       └── main.go          # MCP server entry point — registers tools, calls ServeStdio
├── internal/
│   ├── db/
│   │   └── db.go            # pgxpool connection factory shared by all API handlers
│   └── handler/
│       ├── health.go        # GET /health — DB ping, swag annotations, ErrorResponse/HealthResponse types
│       ├── middleware.go    # SecurityHeaders (OWASP A05) + RequestLogger (OWASP A09)
│       └── tools.go         # POST /tools/calculate, GET /tools/dad-joke — swag annotations, body size cap
├── tools/
│   ├── calculator.go        # MCP tool: pure arithmetic; reference for enum-constrained args
│   ├── calculator_test.go
│   ├── dadjoke.go           # MCP tool: outbound HTTP; reference for RoundTripper injection pattern
│   └── dadjoke_test.go
├── .env.example             # Credential template — copy to .env, never commit .env
├── .golangci.yml            # Linter config: gosec, gocritic, bodyclose, noctx, errcheck
├── .swagignore              # Excludes cmd/api/docs from swag's source scan
├── docker-compose.yml       # Brings up Postgres + API container
├── Dockerfile.api           # Multi-stage build → distroless/static-debian12:nonroot
├── Dockerfile.mcp           # Multi-stage build → scratch with CA certs
├── lefthook.yml             # Git hook definitions
└── mise.toml                # Tool version pins + mise task definitions
```

---

## Running locally

### Option 1 — Everything in Docker

```sh
cp .env.example .env   # edit credentials
docker compose up
```

- API: `http://localhost:8080`
- Postgres: `localhost:5432`

### Option 2 — Postgres in Docker, API on host (faster iteration)

```sh
cp .env.example .env                    # edit credentials
docker compose up postgres -d           # start only Postgres
export DATABASE_URL=postgres://change_me:change_me@localhost:5432/mcp
export SWAGGER_ENABLED=true             # optional — enables /swagger/index.html
go run ./cmd/api
```

### MCP server (for manual testing)

```sh
go run ./cmd/mcp
```

It blocks on stdin — that is correct. Claude Desktop pipes JSON to it; you do not normally run this directly.

---

## REST API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | DB ping — `{"status":"ok"}` or 503 |
| `POST` | `/tools/calculate` | Add / subtract / multiply / divide two numbers |
| `GET` | `/tools/dad-joke` | Proxies a random joke from icanhazdadjoke.com |
| `GET` | `/swagger/*` | Swagger UI (only when `SWAGGER_ENABLED=true`) |

### POST /tools/calculate

```json
// Request
{ "operation": "add", "x": 10, "y": 5 }

// Response 200
{ "result": 15 }

// Response 400 — unknown operation
{ "error": "unknown operation \"mod\": must be add, subtract, multiply, or divide" }

// Response 422 — division by zero
{ "error": "cannot divide by zero" }
```

### GET /tools/dad-joke

```json
// Response 200
{ "id": "R7UfaahVfFd", "joke": "Why do cows wear bells? Because their horns don't work." }
```

---

## Swagger docs

The Swagger spec is pre-generated and committed under `cmd/api/docs/`. After changing any handler annotation, regenerate:

```sh
mise run swagger
# equivalent to: swag init -g cmd/api/main.go -o cmd/api/docs
```

To browse the UI locally:

```sh
SWAGGER_ENABLED=true go run ./cmd/api
open http://localhost:8080/swagger/index.html
```

The spec JSON is also available at `/swagger/doc.json` — useful for importing into JMeter as an OpenAPI test plan.

**Swagger UI is disabled by default** (`SWAGGER_ENABLED=false`). See the Security section for why.

---

## Mise tasks

```sh
mise run swagger   # regenerate swagger docs
mise run build     # go build ./...
mise run test      # go test ./...
mise run lint      # golangci-lint run ./...
```

---

## Claude Desktop config

| OS | Config file |
|----|-------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

**Via `go run`:**
```json
{
  "mcpServers": {
    "mcp-server": {
      "command": "go",
      "args": ["run", "/absolute/path/to/mcp-server/cmd/mcp/main.go"]
    }
  }
}
```

**Via compiled binary:**
```sh
go build -o mcp-server-bin ./cmd/mcp
```
```json
{
  "mcpServers": {
    "mcp-server": {
      "command": "/absolute/path/to/mcp-server-bin"
    }
  }
}
```

Paths must be absolute. Restart Claude Desktop after any config change.

---

## Tests

```sh
go test ./...
```

Also runs automatically on `git push` via lefthook.

---

## Adding a new tool

**`tools/greet.go`:**

```go
package tools

import (
    "context"
    "fmt"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func AddGreet(s *server.MCPServer) {
    tool := mcp.NewTool("greet",
        mcp.WithDescription("Returns a greeting for the given name"),
        mcp.WithString("name", mcp.Required()),
    )
    s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        name, err := req.RequireString("name")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }
        return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
    })
}
```

Register it in `cmd/mcp/main.go`:

```go
tools.AddGreet(s)
```

To expose the same tool over HTTP, add a handler in `internal/handler/` with swag annotations and wire it in `cmd/api/main.go`.

**Go-specific things worth knowing:**

- **Errors are values.** The handler signature is `func(...) (*mcp.CallToolResult, error)`. The `error` return is for unexpected, unrecoverable failures. Tool-level failures — bad input, API errors, divide-by-zero — go into `mcp.NewToolResultError(msg)` with a `nil` error return. This surfaces the failure to the AI as a readable message rather than killing the request.

- **`http.RoundTripper` is the HTTP mock seam.** No mock library needed. Pass an `*http.Client` as a parameter to your internal registration function (see `addDadJokeWithClient` in `tools/dadjoke.go`). In tests, supply a client whose `Transport` is a `roundTripFunc` that redirects to an `httptest.Server`. Production code calls the public `AddDadJoke` wrapper that passes a default client.

- **`_test.go` files in the same package see unexported identifiers.** Tests in `tools/` use `package tools` (not `package tools_test`) so they can call `addDadJokeWithClient` directly. If you need to test internal helpers, keep the test file in the same package.

---

## Tools reference

| Tool | Arguments | Returns |
|------|-----------|---------|
| `calculate` | `operation` (string: `add`/`subtract`/`multiply`/`divide`), `x` (number), `y` (number) | Result to 2 decimal places, e.g. `"25.00"`. Error on divide-by-zero. |
| `dad_joke` | none | Random joke string from [icanhazdadjoke.com](https://icanhazdadjoke.com). Error on network failure or non-200. |

---

## Security (OWASP Top 10)

This section documents each OWASP Top 10 (2021) category, the specific findings in this codebase, and the mitigations applied.

### A01 — Broken Access Control

| Finding | Mitigation |
|---------|-----------|
| No rate limiting; any client can flood endpoints | Per-IP rate limiting via `go-chi/httprate` (60 req/min). Adjust for JMeter runs via the `RATE_LIMIT_RPM` environment variable. |
| Swagger UI exposes full API schema to anyone | Swagger UI is disabled by default (`SWAGGER_ENABLED=false`). Enable only in local dev or a dedicated staging environment. |
| No authentication on tool endpoints | **Known gap — out of scope for this project.** Before exposing this API beyond localhost, add authentication (API key header or OAuth 2.0). See the note below. |

> **Authentication note:** The tool endpoints are intentionally unauthenticated for local JMeter testing. If you expose the API publicly, add an auth middleware (e.g., API key validated from a request header) as the first route-level middleware in `cmd/api/main.go`.

### A02 — Cryptographic Failures

| Finding | Mitigation |
|---------|-----------|
| `docker-compose.yml` previously defaulted to `dev`/`dev` credentials | Variables now use `:?` syntax — Compose exits with an error if they are unset. Copy `.env.example` to `.env` and set strong values. |
| No `.env.example` to guide credential management | `.env.example` added with `openssl rand -base64 32` guidance. `.env` is git-ignored. |
| API container ran as root | `Dockerfile.api` now uses `gcr.io/distroless/static-debian12:nonroot` and `USER nonroot:nonroot` (uid 65532). |
| No TLS enforcement | TLS termination belongs at the reverse proxy (nginx, Caddy, AWS ALB). Document your ingress to terminate TLS before traffic reaches this service. |

### A03 — Injection

| Finding | Mitigation |
|---------|-----------|
| No request body size limit on `POST /tools/calculate` | `http.MaxBytesReader` caps the body at 8 KB before decoding. |
| Non-finite float values could cause undefined arithmetic | Explicit `math.IsNaN` / `math.IsInf` guard added after JSON decode. |
| No SQL queries in current code | The `internal/db` package uses `pgxpool` with parameterised queries only. If you add SQL, always use `$1` placeholders — never `fmt.Sprintf` into a query string. |

### A04 — Insecure Design

| Finding | Mitigation |
|---------|-----------|
| Unbounded request body → DoS vector | 8 KB body cap via `http.MaxBytesReader` (see A03). |
| No input validation beyond JSON decode | Operation validated via exhaustive `switch`; float values validated for finiteness. |

### A05 — Security Misconfiguration

| Finding | Mitigation |
|---------|-----------|
| No HTTP security headers | `handler.SecurityHeaders` middleware sets `X-Content-Type-Options`, `X-Frame-Options: DENY`, `X-XSS-Protection: 0`, `Referrer-Policy`, `Content-Security-Policy`, `Permissions-Policy`, and removes the `Server` header. Applied globally before all handlers. |
| Swagger UI always on | Gated behind `SWAGGER_ENABLED=true` env var (default: `false`). |
| Weak linter coverage | `golangci-lint` now enables `gosec`, `gocritic`, `bodyclose`, `noctx`, and `errcheck`. |

### A06 — Vulnerable and Outdated Components

| Finding | Mitigation |
|---------|-----------|
| Dependencies not audited at build time | `go.sum` ensures cryptographic integrity of every dependency. Run `go mod tidy && go mod verify` to check. |
| Linter didn't catch unclosed HTTP response bodies | `bodyclose` linter now enabled; CI (pre-commit hook) will fail if a response body is not closed. |
| HTTP requests without context propagation | `noctx` linter now enabled; all `http.NewRequest` calls must use `http.NewRequestWithContext`. |

### A07 — Identification and Authentication Failures

No authentication is implemented. See A01 for the known gap and guidance on adding it.

### A08 — Software and Data Integrity Failures

| Finding | Mitigation |
|---------|-----------|
| Build pipeline could be tampered with | Docker images use pinned tags (`postgres:17-alpine`, `golang:1.26-alpine`). `go.sum` verifies module checksums. |

### A09 — Security Logging and Monitoring Failures

| Finding | Mitigation |
|---------|-----------|
| Errors logged without request ID, making correlation impossible | `handler.RequestLogger` middleware emits a structured `slog` line per request with `request_id`, `method`, `path`, `status`, `duration_ms`, `bytes`, and `remote_ip`. Log level is `WARN` for 4xx and `ERROR` for 5xx. |
| No audit trail for client errors | All 4xx responses are logged at `WARN` level, creating an abuse-detection record. |

### A10 — Server-Side Request Forgery (SSRF)

| Finding | Mitigation |
|---------|-----------|
| `GET /tools/dad-joke` makes an outbound HTTP request | The URL (`https://icanhazdadjoke.com/`) is hardcoded — not user-supplied. Not currently exploitable as SSRF. If this endpoint is ever extended to accept a caller-supplied URL, add an allowlist check before making the request. |