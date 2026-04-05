// Package handler contains HTTP handler functions for the REST API.
// Each handler is annotated with swag comments so that swagger docs
// can be generated via `swag init`.
package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthResponse is the response body for the health endpoint.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// ErrorResponse is a generic error response body.
type ErrorResponse struct {
	Error string `json:"error" example:"database unreachable"`
}

// Health returns an HTTP handler that checks database connectivity.
//
//	@Summary		Health check
//	@Description	Returns 200 if the service is healthy and the database is reachable. Returns 503 if the database ping fails.
//	@Tags			ops
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Failure		503	{object}	ErrorResponse
//	@Router			/health [get]
func Health(pool *pgxpool.Pool, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			logger.Error("database ping failed", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, werr := fmt.Fprintf(w, `{"error":"database unreachable"}`); werr != nil {
				logger.Error("failed to write error response", "error", werr)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprintf(w, `{"status":"ok"}`); err != nil {
			logger.Error("failed to write health response", "error", err)
		}
	}
}
