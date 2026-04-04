# mcp-server

An MCP (Model Context Protocol) server written in Go. Ships two example tools — `calculate` and `dad_joke` — covering the two most common patterns: pure logic and an outbound HTTP call.

MCP is a JSON-RPC 2.0 protocol that lets AI clients like Claude call typed tools in an external process. The stdio transport is deliberate: the client spawns your server as a child process and pipes JSON over stdin/stdout. No ports, no auth, no network exposure.

---

## Prerequisites

- **Go 1.26.1+** — `go version`
- **mise** — pins `golangci-lint` and `lefthook` versions from `mise.toml`. Install from [mise.jdx.dev](https://mise.jdx.dev/getting-started.html).

---

## Setup

```sh
mise install          # installs golangci-lint + lefthook at pinned versions
go mod tidy           # fetches dependencies into the module cache
lefthook install      # wires git hooks from lefthook.yml
```

**Git hooks (lefthook.yml):**
- `pre-commit` — runs `go fmt`, `go vet`, `golangci-lint` in parallel; blocks commit on failure
- `pre-push` — runs `go test ./...`

---

## Running the server

```sh
go run main.go
```

It blocks on stdin — that's correct. MCP clients communicate by piping JSON to the process; stray stdout would corrupt the stream. You normally don't run this directly; Claude Desktop launches it as a subprocess.

---

## Claude Desktop config

| OS | Config file |
|----|-------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

**Via `go run` (easiest to iterate):**
```json
{
  "mcpServers": {
    "mcp-server": {
      "command": "go",
      "args": ["run", "/absolute/path/to/mcp-server/main.go"]
    }
  }
}
```

**Via compiled binary (faster startup):**
```sh
go build -o mcp-server-bin .
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

Paths must be absolute. Restart Claude Desktop after any config change.

---

## Tests

```sh
go test ./...
```

Also runs automatically on `git push` via lefthook.

---

## How tool registration works

`main.go` wires everything up before handing off to `ServeStdio`:

```go
s := server.NewMCPServer(
    "mcp-server",
    "0.0.1",
    server.WithToolCapabilities(false),
    server.WithRecovery(), // catches panics in handlers, returns error instead of crashing
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

Then in `main.go`:

```go
tools.AddGreet(s)
```

**Go-specific things worth knowing:**

- **Errors are values.** The handler signature is `func(...) (*mcp.CallToolResult, error)`. The `error` return is for unexpected, unrecoverable failures (framework-level). Tool-level failures — bad input, API errors, divide-by-zero — go into `mcp.NewToolResultError(msg)` with a `nil` error return. This surfaces the failure to the AI as a readable message rather than killing the request.

- **`http.RoundTripper` is the HTTP mock seam.** No mock library needed. Pass an `*http.Client` as a parameter to your internal registration function (see `addDadJokeWithClient` in `dadjoke.go`). In tests, supply a client whose `Transport` is a `roundTripFunc` that redirects to an `httptest.Server`. Production code calls the public `AddDadJoke` wrapper which passes `http.DefaultClient`.

- **`_test.go` files in the same package see unexported identifiers.** Tests in `tools/` use `package tools` (not `package tools_test`) so they can call `addDadJokeWithClient` directly and inject the mock client. If you need to test internal helpers, keep the test file in the same package.

---

## Project structure

```
mcp-server/
├── main.go                  # Server init + tool registration + ServeStdio
├── tools/
│   ├── calculator.go        # Pure arithmetic; reference for enum-constrained args
│   ├── calculator_test.go
│   ├── dadjoke.go           # HTTP call; reference for RoundTripper injection pattern
│   └── dadjoke_test.go
├── go.mod                   # Module: "mcp-server", go 1.26, mcp-go v0.47.0
├── go.sum
├── mise.toml                # Pins golangci-lint + lefthook versions
└── lefthook.yml             # pre-commit: fmt/vet/lint || pre-push: test
```

---

## Tools reference

| Tool | Arguments | Returns |
|------|-----------|---------|
| `calculate` | `operation` (string, one of `add`/`subtract`/`multiply`/`divide`), `x` (number), `y` (number) | Result formatted to 2 decimal places, e.g. `"25.00"`. Error on divide-by-zero. |
| `dad_joke` | none | A random joke string from [icanhazdadjoke.com](https://icanhazdadjoke.com). No API key required. Error on network failure or non-200 response. |