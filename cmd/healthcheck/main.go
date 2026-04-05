// Package main is a minimal HTTP health-check binary used by the Docker
// HEALTHCHECK instruction. It exits 0 when /health returns HTTP 200 and
// 1 otherwise, making it safe to embed in a distroless container image.
package main

import (
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	resp, err := http.Get("http://localhost:" + port + "/health")
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
