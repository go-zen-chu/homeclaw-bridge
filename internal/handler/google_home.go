package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

// ---------------------------------------------------------------------------
// Google Home (Actions SDK) request / response types
// ---------------------------------------------------------------------------

// googleHomeRequest is the webhook payload sent by the Google Actions SDK
// when a user invokes an action on a Google Home device.
type googleHomeRequest struct {
	Handler *googleHomeHandler `json:"handler"`
	Intent  *googleHomeIntent  `json:"intent"`
	Scene   *googleHomeScene   `json:"scene"`
}

type googleHomeHandler struct {
	Name string `json:"name"`
}

type googleHomeIntent struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type googleHomeScene struct {
	Name string `json:"name"`
}

// googleHomeResponse is the fulfillment payload returned to the Actions SDK.
type googleHomeResponse struct {
	Prompt *googleHomePrompt `json:"prompt"`
}

type googleHomePrompt struct {
	Override    bool                     `json:"override"`
	FirstSimple *googleHomeSimpleMessage `json:"firstSimple,omitempty"`
}

type googleHomeSimpleMessage struct {
	Speech string `json:"speech"`
	Text   string `json:"text"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// HandleGoogleHome processes webhook requests from the Google Actions SDK.
// The handler name from the request is mapped to an OpenClaw command, the
// command is forwarded to OpenClaw, and the result is returned as a spoken
// response suitable for Google Home.
//
// Expected request format (POST /google-home):
//
//	{
//	  "handler": { "name": "CheckServerStatus" },
//	  "intent":  { "name": "actions.intent.MAIN" }
//	}
func (h *Handler) HandleGoogleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req googleHomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	handlerName := ""
	if req.Handler != nil {
		handlerName = req.Handler.Name
	}
	if handlerName == "" && req.Intent != nil {
		handlerName = req.Intent.Name
	}
	if handlerName == "" {
		http.Error(w, "missing handler or intent name", http.StatusBadRequest)
		return
	}

	cmdReq := openclaw.CommandRequest{
		Command: handlerName,
	}

	result, err := h.caller.SendCommand(r.Context(), cmdReq)
	if err != nil {
		log.Printf("ERROR: OpenClaw command failed for Google Home request %q: %v", handlerName, err)
		writeJSON(w, http.StatusOK, googleHomeResponse{
			Prompt: &googleHomePrompt{
				Override: false,
				FirstSimple: &googleHomeSimpleMessage{
					Speech: "Sorry, I could not reach the local server. Please try again.",
					Text:   "Sorry, I could not reach the local server. Please try again.",
				},
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, googleHomeResponse{
		Prompt: &googleHomePrompt{
			Override: false,
			FirstSimple: &googleHomeSimpleMessage{
				Speech: result.Message,
				Text:   result.Message,
			},
		},
	})
}
