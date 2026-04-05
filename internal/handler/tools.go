// Package handler contains HTTP handler functions for the REST API.
package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// CalculateRequest is the request body for the calculate endpoint.
type CalculateRequest struct {
	Operation string  `json:"operation" example:"add"`
	X         float64 `json:"x"         example:"10"`
	Y         float64 `json:"y"         example:"5"`
}

// CalculateResponse is the response body for the calculate endpoint.
type CalculateResponse struct {
	Result float64 `json:"result" example:"15"`
}

// DadJokeResponse is the response body for the dad_joke endpoint.
type DadJokeResponse struct {
	ID   string `json:"id"   example:"R7UfaahVfFd"`
	Joke string `json:"joke" example:"Why do cows wear bells? Because their horns don't work."`
}

// Calculate handles arithmetic operations via HTTP.
//
//	@Summary		Calculate a math expression
//	@Description	Performs add, subtract, multiply, or divide on two numbers. Division by zero returns 422.
//	@Tags			tools
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CalculateRequest	true	"Operands and operation"
//	@Success		200		{object}	CalculateResponse
//	@Failure		400		{object}	ErrorResponse	"Invalid request body or unknown operation"
//	@Failure		422		{object}	ErrorResponse	"Division by zero"
//	@Router			/tools/calculate [post]
func Calculate(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CalculateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, logger, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
			return
		}

		var result float64
		switch req.Operation {
		case "add":
			result = req.X + req.Y
		case "subtract":
			result = req.X - req.Y
		case "multiply":
			result = req.X * req.Y
		case "divide":
			if req.Y == 0 {
				writeJSON(w, logger, http.StatusUnprocessableEntity, ErrorResponse{Error: "cannot divide by zero"})
				return
			}
			result = req.X / req.Y
		default:
			writeJSON(w, logger, http.StatusBadRequest, ErrorResponse{
				Error: fmt.Sprintf("unknown operation %q: must be add, subtract, multiply, or divide", req.Operation),
			})
			return
		}

		writeJSON(w, logger, http.StatusOK, CalculateResponse{Result: result})
	}
}

// dadJokeAPIResponse mirrors the JSON shape returned by icanhazdadjoke.com.
type dadJokeAPIResponse struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

// DadJoke fetches a random dad joke from icanhazdadjoke.com and returns it.
//
//	@Summary		Get a random dad joke
//	@Description	Proxies a request to icanhazdadjoke.com and returns the joke and its upstream ID.
//	@Tags			tools
//	@Produce		json
//	@Success		200	{object}	DadJokeResponse
//	@Failure		502	{object}	ErrorResponse	"Upstream request failed"
//	@Router			/tools/dad-joke [get]
func DadJoke(logger *slog.Logger) http.HandlerFunc {
	client := &http.Client{Timeout: 10 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://icanhazdadjoke.com/", nil)
		if err != nil {
			logger.Error("failed to build upstream request", "error", err)
			writeJSON(w, logger, http.StatusBadGateway, ErrorResponse{Error: "failed to build upstream request"})
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "mcp-server/api")

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("upstream dad joke request failed", "error", err)
			writeJSON(w, logger, http.StatusBadGateway, ErrorResponse{Error: "upstream request failed"})
			return
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				logger.Error("failed to close upstream response body", "error", cerr)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			logger.Error("unexpected upstream status", "status", resp.StatusCode)
			writeJSON(w, logger, http.StatusBadGateway, ErrorResponse{
				Error: fmt.Sprintf("upstream returned status %d", resp.StatusCode),
			})
			return
		}

		var upstream dadJokeAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
			logger.Error("failed to decode upstream response", "error", err)
			writeJSON(w, logger, http.StatusBadGateway, ErrorResponse{Error: "failed to decode upstream response"})
			return
		}

		writeJSON(w, logger, http.StatusOK, DadJokeResponse{
			ID:   upstream.ID,
			Joke: upstream.Joke,
		})
	}
}

// writeJSON marshals v as JSON and writes it to w with the given status code.
// Any marshalling or write error is logged but not returned to the caller.
func writeJSON(w http.ResponseWriter, logger *slog.Logger, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("failed to write JSON response", "error", err)
	}
}
