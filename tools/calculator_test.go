// Package tools — calculator_test.go
//
// Tests for the calculator tool. Strategy: create an MCP server, register the
// tool, retrieve its handler via s.GetTool, and invoke it directly — no network
// required.

package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newCalcServer creates a fresh MCP server with the calculator tool registered.
func newCalcServer(t *testing.T) *server.MCPServer {
	t.Helper()
	s := server.NewMCPServer("test-server", "0.0.0")
	AddCalculator(s)
	return s
}

// callCalc looks up the "calculate" tool, builds a request from the supplied
// arguments, and invokes the handler.
func callCalc(t *testing.T, s *server.MCPServer, op string, x, y float64) *mcp.CallToolResult {
	t.Helper()

	tool := s.GetTool("calculate")
	if tool == nil {
		t.Fatal("the 'calculate' tool was not found on the server — did AddCalculator run?")
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calculate",
			Arguments: map[string]any{
				"operation": op,
				"x":         x,
				"y":         y,
			},
		},
	}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		// The calculator surfaces errors via result.IsError, not as a Go error.
		// A non-nil err here indicates a framework bug or escaped panic.
		t.Fatalf("handler returned an unexpected Go error: %v", err)
	}
	return result
}

// resultText extracts the plain-text string from the first content item of a result.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("result.Content is empty — the handler returned no content items")
	}

	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected result.Content[0] to be mcp.TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

// ---------------------------------------------------------------------------
// Happy-path tests
// ---------------------------------------------------------------------------

// TestCalculator_Add: add(3, 4) → "7.00"
func TestCalculator_Add(t *testing.T) {
	s := newCalcServer(t)
	result := callCalc(t, s, "add", 3, 4)

	if result.IsError {
		t.Fatalf("add(3, 4) unexpectedly returned an error: %s", resultText(t, result))
	}

	got := resultText(t, result)
	want := "7.00"
	if got != want {
		t.Errorf("add(3, 4) = %q, want %q", got, want)
	}
}

// TestCalculator_Subtract: subtract(10, 3) → "7.00"
func TestCalculator_Subtract(t *testing.T) {
	s := newCalcServer(t)
	result := callCalc(t, s, "subtract", 10, 3)

	if result.IsError {
		t.Fatalf("subtract(10, 3) unexpectedly returned an error: %s", resultText(t, result))
	}

	got := resultText(t, result)
	want := "7.00"
	if got != want {
		t.Errorf("subtract(10, 3) = %q, want %q", got, want)
	}
}

// TestCalculator_Multiply: multiply(6, 7) → "42.00"
func TestCalculator_Multiply(t *testing.T) {
	s := newCalcServer(t)
	result := callCalc(t, s, "multiply", 6, 7)

	if result.IsError {
		t.Fatalf("multiply(6, 7) unexpectedly returned an error: %s", resultText(t, result))
	}

	got := resultText(t, result)
	want := "42.00"
	if got != want {
		t.Errorf("multiply(6, 7) = %q, want %q", got, want)
	}
}

// TestCalculator_Divide: divide(7, 2) → "3.50"
func TestCalculator_Divide(t *testing.T) {
	s := newCalcServer(t)
	result := callCalc(t, s, "divide", 7, 2)

	if result.IsError {
		t.Fatalf("divide(7, 2) unexpectedly returned an error: %s", resultText(t, result))
	}

	got := resultText(t, result)
	want := "3.50"
	if got != want {
		t.Errorf("divide(7, 2) = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Error-case tests
// ---------------------------------------------------------------------------

// TestCalculator_DivideByZero: divide(5, 0) must return an error result, not
// +Inf. float64 division by zero doesn't panic in Go, so the handler guards
// against it explicitly.
func TestCalculator_DivideByZero(t *testing.T) {
	s := newCalcServer(t)
	result := callCalc(t, s, "divide", 5, 0)

	// Errors are surfaced via IsError=true on the result, not as a Go error,
	// so the MCP protocol layer remains unaware of tool-level failures.
	if !result.IsError {
		t.Fatalf("divide(5, 0) should have returned an error result, but IsError is false; got %q", resultText(t, result))
	}

	got := resultText(t, result)
	want := "cannot divide by zero"
	if got != want {
		t.Errorf("divide-by-zero error message = %q, want %q", got, want)
	}
}

// TestCalculator_MissingArgument: omitting a required argument must yield
// IsError=true rather than a panic or garbage value.
func TestCalculator_MissingArgument(t *testing.T) {
	s := newCalcServer(t)

	tool := s.GetTool("calculate")
	if tool == nil {
		t.Fatal("the 'calculate' tool was not found on the server — did AddCalculator run?")
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calculate",
			Arguments: map[string]any{
				"operation": "add",
				"x":         float64(1),
				// "y" intentionally omitted
			},
		},
	}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned an unexpected Go error: %v", err)
	}

	if !result.IsError {
		t.Fatalf("expected IsError=true when 'y' is missing, but got a success result: %q", resultText(t, result))
	}

	// Don't assert the exact message — it's generated by mcp-go internals.
	if msg := resultText(t, result); msg == "" {
		t.Error("expected a non-empty error message when a required argument is missing")
	}
}
