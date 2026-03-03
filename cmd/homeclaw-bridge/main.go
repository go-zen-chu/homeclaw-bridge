// Package main is the entry point for the homeclaw-bridge server.
// It starts a local HTTP server that bridges smart speaker requests
// (Google Home, Alexa) to the OpenClaw server on the local network.
//
// Configuration is done via environment variables:
//
//	LISTEN_ADDR   — address to listen on (default ":8080")
//	OPENCLAW_URL  — base URL of the OpenClaw server (default "http://localhost:9090")
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-zen-chu/homeclaw-bridge/internal/handler"
	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

func main() {
	addr := envOrDefault("LISTEN_ADDR", ":8080")
	openClawURL := envOrDefault("OPENCLAW_URL", "http://localhost:9090")

	client := openclaw.NewClient(openClawURL)
	h := handler.New(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/google-home", h.HandleGoogleHome)
	mux.HandleFunc("/alexa", h.HandleAlexa)
	mux.HandleFunc("/health", h.HandleHealth)

	log.Printf("homeclaw-bridge listening on %s", addr)
	log.Printf("forwarding requests to OpenClaw at %s", openClawURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
