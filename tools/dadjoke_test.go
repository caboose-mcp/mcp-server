// Package tools — dadjoke_test.go
//
// Tests for the dad_joke tool. Uses httptest + a roundTripFunc transport to
// intercept outbound requests and redirect them to a local test server,
// avoiding real network calls and allowing controlled error simulation.

package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

// roundTripFunc adapts a function to the http.RoundTripper interface.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// newDadJokeClient starts a local httptest.Server with the given handler and
// returns an *http.Client whose transport rewrites every request to point at
// it, plus a cleanup func.
//
// Why rewrite the URL?
//
//	The tool handler has "https://icanhazdadjoke.com/" hardcoded. We intercept
//	at the transport layer to swap the scheme + host for the test server's
//	address rather than trying to parameterise the URL through the handler.
func newDadJokeClient(t *testing.T, handler http.HandlerFunc) (*http.Client, func()) {
	t.Helper()

	ts := httptest.NewServer(handler)

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			serverURL, err := url.Parse(ts.URL)
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.URL.Scheme = serverURL.Scheme
			req.URL.Host = serverURL.Host
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	return client, ts.Close
}

// invokeDadJoke registers the dad_joke tool with the given client and calls
// its handler, returning the result.
func invokeDadJoke(t *testing.T, client *http.Client) *mcp.CallToolResult {
	t.Helper()

	s := server.NewMCPServer("test-server", "0.0.0")
	addDadJokeWithClient(s, client)

	tool := s.GetTool("dad_joke")
	if tool == nil {
		t.Fatal("the 'dad_joke' tool was not found on the server — did addDadJokeWithClient register it?")
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "dad_joke",
			Arguments: map[string]any{},
		},
	}

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned an unexpected Go error: %v", err)
	}
	return result
}

// ---------------------------------------------------------------------------
// Test functions
// ---------------------------------------------------------------------------

// TestDadJoke_HappyPath: HTTP 200 with valid JSON → IsError=false, result text
// equals the joke string. Also verifies the Accept: application/json header is
// sent (required by the API to receive JSON rather than plain text).
func TestDadJoke_HappyPath(t *testing.T) {
	const wantJoke = "Why don't scientists trust atoms? Because they make up everything!"

	client, cleanup := newDadJokeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept header = %q, want \"application/json\"", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"id":"test-id-001","joke":"` + wantJoke + `","status":200}`)); err != nil {
			t.Errorf("failed to write response body: %v", err)
		}
	})
	defer cleanup()

	result := invokeDadJoke(t, client)

	if result.IsError {
		t.Fatalf("happy-path request unexpectedly returned IsError=true; message: %s",
			resultText(t, result))
	}

	if got := resultText(t, result); got != wantJoke {
		t.Errorf("joke text = %q, want %q", got, wantJoke)
	}
}

// TestDadJoke_NonOKStatus: non-200 response → IsError=true, message contains
// the status code.
func TestDadJoke_NonOKStatus(t *testing.T) {
	client, cleanup := newDadJokeClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer cleanup()

	result := invokeDadJoke(t, client)

	if !result.IsError {
		t.Fatalf("expected IsError=true for a 503 response, but got a success result: %q",
			resultText(t, result))
	}

	if msg := resultText(t, result); !strings.Contains(msg, "503") {
		t.Errorf("error message %q should contain the HTTP status code \"503\"", msg)
	}
}

// TestDadJoke_BadJSON: HTTP 200 with malformed body → IsError=true, non-empty
// error message.
func TestDadJoke_BadJSON(t *testing.T) {
	client, cleanup := newDadJokeClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{this is : not [ valid json!!!`)); err != nil {
			t.Errorf("failed to write response body: %v", err)
		}
	})
	defer cleanup()

	result := invokeDadJoke(t, client)

	if !result.IsError {
		t.Fatalf("expected IsError=true for malformed JSON, but got a success result: %q",
			resultText(t, result))
	}

	if msg := resultText(t, result); msg == "" {
		t.Error("error message should not be empty when JSON decoding fails")
	}
}
