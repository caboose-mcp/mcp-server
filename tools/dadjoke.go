package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// dadJokeResponse mirrors the JSON shape returned by icanhazdadjoke.com.
type dadJokeResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

// addDadJokeWithClient registers the dad_joke tool on s using the given HTTP
// client, allowing tests to inject a transport that targets a local server.
func addDadJokeWithClient(s *server.MCPServer, client *http.Client) {
	tool := mcp.NewTool("dad_joke",
		mcp.WithDescription("Fetches a random dad joke"),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://icanhazdadjoke.com/", nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to build request: %v", err)), nil
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "mcp-server (github.com/mark3labs/mcp-go)")

		resp, err := client.Do(req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch joke: %v", err)), nil
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "failed to close response body:", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return mcp.NewToolResultError(fmt.Sprintf("unexpected status: %d", resp.StatusCode)), nil
		}

		var joke dadJokeResponse
		if err := json.NewDecoder(resp.Body).Decode(&joke); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to decode response: %v", err)), nil
		}

		return mcp.NewToolResultText(joke.Joke), nil
	})
}

func AddDadJoke(s *server.MCPServer) {
	addDadJokeWithClient(s, &http.Client{Timeout: 10 * time.Second})
}
