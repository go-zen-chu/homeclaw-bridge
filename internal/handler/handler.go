// Package handler provides HTTP handlers for smart speaker requests.
// It supports Google Home (Actions SDK) and Alexa (Custom Skill) local fulfillment,
// and bridges every recognised command to the OpenClaw server on the local network.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

// Handler holds the OpenClaw caller used by all request handlers.
type Handler struct {
	caller openclaw.Caller
}

// New creates a Handler backed by the provided OpenClaw caller.
func New(caller openclaw.Caller) *Handler {
	return &Handler{caller: caller}
}

// HandleHealth returns 200 OK so that infrastructure can liveness-probe the server.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// writeJSON is a helper that serialises v as JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
