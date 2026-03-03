package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zen-chu/homeclaw-bridge/internal/handler"
	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

// ---------------------------------------------------------------------------
// Fake OpenClaw caller for tests
// ---------------------------------------------------------------------------

type fakeCaller struct {
	response *openclaw.CommandResponse
	err      error
	lastCmd  openclaw.CommandRequest
}

func (f *fakeCaller) SendCommand(_ context.Context, cmd openclaw.CommandRequest) (*openclaw.CommandResponse, error) {
	f.lastCmd = cmd
	return f.response, f.err
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

func TestHandleHealth(t *testing.T) {
	h := handler.New(&fakeCaller{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

// ---------------------------------------------------------------------------
// Google Home handler
// ---------------------------------------------------------------------------

func TestHandleGoogleHome_Success(t *testing.T) {
	fc := &fakeCaller{response: &openclaw.CommandResponse{Status: "ok", Message: "Server is running"}}
	h := handler.New(fc)

	payload := `{"handler":{"name":"CheckServerStatus"},"intent":{"name":"actions.intent.MAIN"}}`
	req := httptest.NewRequest(http.MethodPost, "/google-home", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if fc.lastCmd.Command != "CheckServerStatus" {
		t.Errorf("expected command CheckServerStatus, got %q", fc.lastCmd.Command)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	prompt, ok := resp["prompt"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected prompt object, got %T", resp["prompt"])
	}
	simple, ok := prompt["firstSimple"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected firstSimple object, got %T", prompt["firstSimple"])
	}
	if simple["text"] != "Server is running" {
		t.Errorf("expected text 'Server is running', got %q", simple["text"])
	}
}

func TestHandleGoogleHome_FallsBackToIntentName(t *testing.T) {
	fc := &fakeCaller{response: &openclaw.CommandResponse{Status: "ok", Message: "done"}}
	h := handler.New(fc)

	// No "handler" key — should fall back to intent name.
	payload := `{"intent":{"name":"RestartService"}}`
	req := httptest.NewRequest(http.MethodPost, "/google-home", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if fc.lastCmd.Command != "RestartService" {
		t.Errorf("expected command RestartService, got %q", fc.lastCmd.Command)
	}
}

func TestHandleGoogleHome_OpenClawError_ReturnsFriendlyMessage(t *testing.T) {
	fc := &fakeCaller{err: context.DeadlineExceeded}
	h := handler.New(fc)

	payload := `{"handler":{"name":"CheckServerStatus"}}`
	req := httptest.NewRequest(http.MethodPost, "/google-home", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	// Even on error the response must be 200 so Google Home can read the message.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	prompt := resp["prompt"].(map[string]interface{})
	simple := prompt["firstSimple"].(map[string]interface{})
	if simple["text"] == "" {
		t.Error("expected a non-empty error message in the response")
	}
}

func TestHandleGoogleHome_MethodNotAllowed(t *testing.T) {
	h := handler.New(&fakeCaller{})
	req := httptest.NewRequest(http.MethodGet, "/google-home", nil)
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleGoogleHome_InvalidJSON(t *testing.T) {
	h := handler.New(&fakeCaller{})
	req := httptest.NewRequest(http.MethodPost, "/google-home", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGoogleHome_MissingHandlerAndIntent(t *testing.T) {
	h := handler.New(&fakeCaller{})
	req := httptest.NewRequest(http.MethodPost, "/google-home", bytes.NewBufferString("{}"))
	rec := httptest.NewRecorder()
	h.HandleGoogleHome(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Alexa handler
// ---------------------------------------------------------------------------

func TestHandleAlexa_Success(t *testing.T) {
	fc := &fakeCaller{response: &openclaw.CommandResponse{Status: "ok", Message: "Server is running"}}
	h := handler.New(fc)

	payload := `{
		"version":"1.0",
		"session":{"sessionId":"abc"},
		"request":{"type":"IntentRequest","requestId":"req1","intent":{"name":"CheckServerStatusIntent"}}
	}`
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if fc.lastCmd.Command != "CheckServerStatusIntent" {
		t.Errorf("expected command CheckServerStatusIntent, got %q", fc.lastCmd.Command)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	response := resp["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] != "Server is running" {
		t.Errorf("expected text 'Server is running', got %q", speech["text"])
	}
}

func TestHandleAlexa_WithSlots(t *testing.T) {
	fc := &fakeCaller{response: &openclaw.CommandResponse{Status: "ok", Message: "Restarted"}}
	h := handler.New(fc)

	payload := `{
		"version":"1.0",
		"request":{
			"type":"IntentRequest",
			"intent":{
				"name":"RestartServiceIntent",
				"slots":{"service":{"name":"service","value":"database"}}
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if fc.lastCmd.Params["service"] != "database" {
		t.Errorf("expected slot service=database, got %q", fc.lastCmd.Params["service"])
	}
}

func TestHandleAlexa_LaunchRequest_WelcomeMessage(t *testing.T) {
	h := handler.New(&fakeCaller{})

	payload := `{"version":"1.0","request":{"type":"LaunchRequest"}}`
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	response := resp["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] == "" {
		t.Error("expected a non-empty welcome message")
	}
}

func TestHandleAlexa_OpenClawError_ReturnsFriendlyMessage(t *testing.T) {
	fc := &fakeCaller{err: context.DeadlineExceeded}
	h := handler.New(fc)

	payload := `{"version":"1.0","request":{"type":"IntentRequest","intent":{"name":"CheckServerStatusIntent"}}}`
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	// Even on error the HTTP status must be 200 so Alexa can read the spoken message.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	response := resp["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestHandleAlexa_MethodNotAllowed(t *testing.T) {
	h := handler.New(&fakeCaller{})
	req := httptest.NewRequest(http.MethodGet, "/alexa", nil)
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAlexa_InvalidJSON(t *testing.T) {
	h := handler.New(&fakeCaller{})
	req := httptest.NewRequest(http.MethodPost, "/alexa", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	h.HandleAlexa(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
