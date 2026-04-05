package main

import (
	"fmt"
	"os"

	"mcp-server/tools"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"mcp-server",
		"0.0.1",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	tools.AddCalculator(s)
	tools.AddDadJoke(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
	}
}
