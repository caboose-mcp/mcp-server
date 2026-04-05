// Package handler contains HTTP handler functions and middleware for the REST API.
package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// SecurityHeaders returns middleware that sets defensive HTTP response headers
// on every response. These headers address several OWASP Top 10 categories:
//
//   - A05 Security Misconfiguration: prevents MIME-type sniffing, clickjacking,
//     and information leakage via the Server header.
//   - A03 Injection: Content-Security-Policy blocks inline script execution,
//     mitigating reflected XSS if HTML is ever served.
//
// None of these headers require knowledge of the response body, so they are
// applied unconditionally before the next handler runs.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Prevent browsers from MIME-sniffing the Content-Type away from the
		// declared value. Required by OWASP and PCI-DSS.
		h.Set("X-Content-Type-Options", "nosniff")

		// Block the page from being embedded in a frame. Mitigates clickjacking
		// (OWASP A05). DENY is stricter than SAMEORIGIN and appropriate for an API.
		h.Set("X-Frame-Options", "DENY")

		// Modern recommendation is to set this to "0" to disable the legacy IE
		// XSS filter, which itself introduced vulnerabilities in some browsers.
		h.Set("X-XSS-Protection", "0")

		// Only send the origin portion of the Referrer header on cross-origin
		// requests to avoid leaking path/query parameters to third parties.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrictive CSP for an API that never intentionally serves HTML.
		// The Swagger UI bootstrap relies on an inline <script> block, so
		// routes under /swagger/ receive a relaxed policy that permits inline
		// scripts; all other routes keep the strict policy.
		if strings.HasPrefix(r.URL.Path, "/swagger/") {
			h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self' data:")
		} else {
			h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		}

		// Disable all browser feature APIs for responses from this origin.
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Remove the Server header to avoid advertising implementation details.
		// Go's net/http sets "Go-http-client" on some responses; clearing it
		// here ensures it never reaches the client.
		h.Del("Server")

		next.ServeHTTP(w, r)
	})
}

// responseRecorder wraps http.ResponseWriter to capture the status code after
// the handler has written it. chi's middleware.WrapResponseWriter provides a
// similar facility, but keeping this local avoids an extra import and makes the
// intent explicit.
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	n, err := rr.ResponseWriter.Write(b)
	rr.bytes += n
	return n, err
}

// RequestLogger returns middleware that emits a structured slog log line for
// every completed request. It addresses OWASP A09 (Security Logging and
// Monitoring Failures) by recording:
//
//   - request_id  — propagated from chi's RequestID middleware so log lines
//     can be correlated across services.
//   - method / path / status — the essential HTTP fields.
//   - duration_ms — latency for performance monitoring.
//   - bytes       — response size.
//   - remote_ip   — sourced from chi's RealIP middleware (X-Real-IP /
//     X-Forwarded-For when trusted, otherwise RemoteAddr).
//
// Log level is chosen based on status:
//
//	5xx → Error   (service fault, needs immediate attention)
//	4xx → Warn    (client error, useful for abuse detection)
//	otherwise → Info
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rr := &responseRecorder{
				ResponseWriter: w,
				// Default to 200 so that handlers that never call WriteHeader
				// (which is valid — net/http infers 200) are recorded correctly.
				status: http.StatusOK,
			}

			next.ServeHTTP(rr, r)

			duration := time.Since(start)
			reqID := middleware.GetReqID(r.Context())

			attrs := []any{
				slog.String("request_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rr.status),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.Int("bytes", rr.bytes),
				slog.String("remote_ip", r.RemoteAddr),
			}

			switch {
			case rr.status >= 500:
				logger.Error("request completed", attrs...)
			case rr.status >= 400:
				logger.Warn("request completed", attrs...)
			default:
				logger.Info("request completed", attrs...)
			}
		})
	}
}
