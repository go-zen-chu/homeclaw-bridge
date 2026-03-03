package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

// ---------------------------------------------------------------------------
// Alexa Custom Skill request / response types
// ---------------------------------------------------------------------------

// alexaRequest is the webhook payload sent by the Alexa Skills Kit when a
// user invokes a custom skill on an Alexa device.
type alexaRequest struct {
	Version string       `json:"version"`
	Session alexaSession `json:"session"`
	Request alexaBody    `json:"request"`
}

type alexaSession struct {
	SessionID string `json:"sessionId"`
}

type alexaBody struct {
	Type      string      `json:"type"`
	RequestID string      `json:"requestId"`
	Intent    alexaIntent `json:"intent,omitempty"`
}

type alexaIntent struct {
	Name  string                  `json:"name"`
	Slots map[string]alexaSlot    `json:"slots,omitempty"`
}

type alexaSlot struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// alexaResponse is the fulfillment payload returned to the Alexa Skills Kit.
type alexaResponse struct {
	Version  string         `json:"version"`
	Response alexaRespBody  `json:"response"`
}

type alexaRespBody struct {
	OutputSpeech     *alexaOutputSpeech `json:"outputSpeech,omitempty"`
	ShouldEndSession bool               `json:"shouldEndSession"`
}

type alexaOutputSpeech struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// HandleAlexa processes webhook requests from the Alexa Skills Kit.
// The intent name from the request is mapped to an OpenClaw command, the
// command is forwarded to OpenClaw, and the result is returned as a spoken
// response suitable for Alexa.
//
// Expected request format (POST /alexa):
//
//	{
//	  "version": "1.0",
//	  "session": { "sessionId": "..." },
//	  "request": {
//	    "type": "IntentRequest",
//	    "requestId": "...",
//	    "intent": { "name": "CheckServerStatusIntent" }
//	  }
//	}
func (h *Handler) HandleAlexa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req alexaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	intentName := req.Request.Intent.Name
	if intentName == "" {
		// For LaunchRequest or SessionEndedRequest we have no intent to forward.
		slog.Info("alexa launch or session-ended request received",
			"remote_addr", r.RemoteAddr,
			"request_type", req.Request.Type,
		)
		writeJSON(w, http.StatusOK, alexaResponse{
			Version: "1.0",
			Response: alexaRespBody{
				OutputSpeech: &alexaOutputSpeech{
					Type: "PlainText",
					Text: "Welcome to HomeClaw Bridge. You can ask me to check the server status.",
				},
				ShouldEndSession: false,
			},
		})
		return
	}

	slog.Info("alexa intent request received",
		"remote_addr", r.RemoteAddr,
		"intent_name", intentName,
	)

	cmdReq := openclaw.CommandRequest{
		Command: intentName,
	}

	// Forward any slot values as command parameters.
	if len(req.Request.Intent.Slots) > 0 {
		cmdReq.Params = make(map[string]string, len(req.Request.Intent.Slots))
		for k, slot := range req.Request.Intent.Slots {
			cmdReq.Params[k] = slot.Value
		}
	}

	result, err := h.caller.SendCommand(r.Context(), cmdReq)
	if err != nil {
		slog.Error("openclaw command failed for alexa intent",
			"intent_name", intentName,
			"err", err,
		)
		writeJSON(w, http.StatusOK, alexaResponse{
			Version: "1.0",
			Response: alexaRespBody{
				OutputSpeech: &alexaOutputSpeech{
					Type: "PlainText",
					Text: "Sorry, I could not reach the local server. Please try again.",
				},
				ShouldEndSession: true,
			},
		})
		return
	}

	slog.Info("alexa intent request completed",
		"intent_name", intentName,
		"response_message", result.Message,
	)
	writeJSON(w, http.StatusOK, alexaResponse{
		Version: "1.0",
		Response: alexaRespBody{
			OutputSpeech: &alexaOutputSpeech{
				Type: "PlainText",
				Text: result.Message,
			},
			ShouldEndSession: true,
		},
	})
}
