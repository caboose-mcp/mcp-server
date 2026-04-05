# mcp-server

A monorepo containing two services: an MCP server and a REST API, sharing a common Postgres database.

MCP (Model Context Protocol) is a JSON-RPC 2.0 protocol for exposing typed tools to AI clients like Claude. The MCP server uses stdio transport — Claude Desktop spawns it as a child process and communicates over stdin/stdout. No ports, no auth, no network exposure.

---

## Services

| Service | Entry point | Transport |
|---------|-------------|-----------|
| MCP server | `cmd/mcp` | stdio (spawned by Claude Desktop) |
| REST API | `cmd/api` | HTTP on `PORT` (default `8080`) |

---

## Prerequisites

- **Go 1.26.1+**
- **mise** — manages `golangci-lint` and `lefthook` at pinned versions. Install from [mise.jdx.dev](https://mise.jdx.dev/getting-started.html).
- **Docker + Docker Compose** — for running the API and Postgres locally.

---

## Setup

```sh
mise install          # installs golangci-lint + lefthook at pinned versions
go mod tidy           # fetches Go dependencies
lefthook install      # wires git hooks from lefthook.yml
cp .env.example .env  # copy and edit environment variables
```

**Git hooks (`lefthook.yml`):**
- `pre-commit` — runs `go fmt`, `go vet`, `golangci-lint` in parallel; blocks commit on failure
- `pre-push` — runs `go test ./...`

---

## Running locally

**API + Postgres via Docker Compose:**
```sh
docker compose up
```

The API will be available at `http://localhost:8080`. Postgres is exposed on `localhost:5432`.

**MCP server (for manual testing):**
```sh
go run ./cmd/mcp
```
It blocks on stdin — that's expected. In normal use Claude Desktop launches it; you don't run it directly.

---

## Environment variables

Defined in `.env` (copy from `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `dev` | Postgres username |
| `POSTGRES_PASSWORD` | `dev` | Postgres password |
| `POSTGRES_DB` | `dev` | Postgres database name |
| `DATABASE_URL` | — | Full DSN, set automatically by Compose for the API container |
| `PORT` | `8080` | Port the REST API listens on |

---

## Claude Desktop config

| OS | Config file |
|----|-------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

**Via `go run` (easiest during development):**
```json
{
  "mcpServers": {
    "mcp-server": {
      "command": "go",
      "args": ["run", "/absolute/path/to/mcp-server/cmd/mcp"]
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
      "command": "/absolute/path/to/mcp-server/mcp-server-bin"
    }
  }
}
```

**Via Docker:**
```sh
docker build -f Dockerfile.mcp -t mcp-server-mcp .
```
```json
{
  "mcpServers": {
    "mcp-server": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "mcp-server-mcp"]
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

## How tool registration works

`cmd/mcp/main.go` wires everything up before handing off to `ServeStdio`:

```go
s := server.NewMCPServer(
    "mcp-server",
    "0.0.1",
    server.WithToolCapabilities(false),
    server.WithRecovery(), // catches panics in handlers, returns error result instead of crashing
)

tools.AddCalculator(s)
tools.AddDadJoke(s)

server.ServeStdio(s) // blocks; reads JSON-RPC from stdin, writes to stdout
```

Each tool lives in `tools/` and exposes a single `AddXxx(s *server.MCPServer)` function. `main.go` only calls `Add*` functions — one per tool, no other coupling.

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

Then in `cmd/mcp/main.go`:
```go
tools.AddGreet(s)
```

**Go-specific things worth knowing:**

- **Errors are values.** The handler returns `(*mcp.CallToolResult, error)`. The `error` return is for framework-level failures. Tool-level failures — bad input, API errors, divide-by-zero — go into `mcp.NewToolResultError(msg)` with a `nil` error return. This surfaces the failure to the AI as a readable message rather than killing the request.

- **`http.RoundTripper` is the HTTP mock seam.** No mock library needed. Pass an `*http.Client` into your internal registration function (see `addDadJokeWithClient` in `tools/dadjoke.go`). In tests, supply a client whose `Transport` redirects to an `httptest.Server`. The public `AddDadJoke` wrapper passes `http.DefaultClient`.

- **`_test.go` files in the same package see unexported identifiers.** Tests in `tools/` use `package tools` (not `package tools_test`) so they can call `addDadJokeWithClient` directly to inject the mock client.

---

## Project structure

```
mcp-server/
├── cmd/
│   ├── mcp/
│   │   └── main.go          # MCP server: init, tool registration, ServeStdio
│   └── api/
│       └── main.go          # REST API: chi router, graceful shutdown, DB wiring
├── internal/
│   └── db/
│       └── db.go            # pgxpool connection helper, shared by both services
├── tools/
│   ├── calculator.go        # Pure arithmetic; reference for enum-constrained args
│   ├── calculator_test.go
│   ├── dadjoke.go           # Outbound HTTP; reference for RoundTripper injection pattern
│   └── dadjoke_test.go
├── Dockerfile.mcp           # Multi-stage build → scratch image (~5MB)
├── Dockerfile.api           # Multi-stage build → distroless image
├── docker-compose.yml       # postgres + api
├── .env.example             # Copy to .env for local development
├── .golangci.yml            # Enables gosec (OWASP) and gocritic on top of defaults
├── go.mod                   # Module: "mcp-server", go 1.26.1, mcp-go v0.47.0
├── go.sum
├── mise.toml                # Pins golangci-lint and lefthook to exact versions
└── lefthook.yml             # pre-commit: fmt/vet/lint in parallel | pre-push: test
```

---

## MCP tools reference

| Tool | Arguments | Returns |
|------|-----------|---------|
| `calculate` | `operation` (enum: `add`/`subtract`/`multiply`/`divide`), `x` (number), `y` (number) | Result to 2 decimal places. Error on divide-by-zero. |
| `dad_joke` | none | Random joke from [icanhazdadjoke.com](https://icanhazdadjoke.com). No API key required. Error on network failure or non-200 response. |

## REST API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Returns `200 ok` if the server is up and Postgres is reachable; `503` otherwise. |