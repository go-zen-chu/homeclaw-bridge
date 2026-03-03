// Package e2e contains end-to-end tests for homeclaw-bridge.
//
// Each test starts a real httptest.Server for homeclaw-bridge and a separate
// httptest.Server that fakes the OpenClaw backend. Requests flow through the
// full HTTP stack so that routing, JSON parsing, and round-trip forwarding are
// all exercised.
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zen-chu/homeclaw-bridge/internal/handler"
	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

// startFakeOpenClaw creates an httptest server that records the last command it
// received and returns the given response message.
func startFakeOpenClaw(t *testing.T, responseMessage string) (*httptest.Server, *openclaw.CommandRequest) {
	t.Helper()
	var last openclaw.CommandRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&last); err != nil {
			t.Errorf("fake openclaw: decoding request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openclaw.CommandResponse{
			Status:  "ok",
			Message: responseMessage,
		})
	}))
	return srv, &last
}

// startBridge creates an httptest.Server running the real homeclaw-bridge mux
// backed by the OpenClaw client pointing at openClawURL.
func startBridge(openClawURL string) *httptest.Server {
	client := openclaw.NewClient(openClawURL)
	h := handler.New(client)

	mux := http.NewServeMux()
	mux.HandleFunc("/google-home", h.HandleGoogleHome)
	mux.HandleFunc("/alexa", h.HandleAlexa)
	mux.HandleFunc("/health", h.HandleHealth)

	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

func TestE2E_Health(t *testing.T) {
	bridge := startBridge("http://127.0.0.1:1") // openclaw URL unused for /health
	defer bridge.Close()

	resp, err := http.Get(bridge.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

// ---------------------------------------------------------------------------
// Google Home — full round-trip
// ---------------------------------------------------------------------------

func TestE2E_GoogleHome_FullRoundTrip(t *testing.T) {
	fakeOC, lastCmd := startFakeOpenClaw(t, "Server is running fine")
	defer fakeOC.Close()

	bridge := startBridge(fakeOC.URL)
	defer bridge.Close()

	payload := `{"handler":{"name":"CheckServerStatus"},"intent":{"name":"actions.intent.MAIN"}}`
	resp, err := http.Post(bridge.URL+"/google-home", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /google-home: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if lastCmd.Command != "CheckServerStatus" {
		t.Errorf("expected OpenClaw to receive command CheckServerStatus, got %q", lastCmd.Command)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	prompt := body["prompt"].(map[string]interface{})
	simple := prompt["firstSimple"].(map[string]interface{})
	if simple["text"] != "Server is running fine" {
		t.Errorf("unexpected speech text: %q", simple["text"])
	}
}

func TestE2E_GoogleHome_OpenClawDown_ReturnsFriendlyMessage(t *testing.T) {
	// Point bridge at a URL where nothing is listening.
	bridge := startBridge("http://127.0.0.1:1")
	defer bridge.Close()

	payload := `{"handler":{"name":"CheckServerStatus"}}`
	resp, err := http.Post(bridge.URL+"/google-home", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /google-home: %v", err)
	}
	defer resp.Body.Close()

	// Bridge must return 200 even when OpenClaw is unreachable.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	prompt := body["prompt"].(map[string]interface{})
	simple := prompt["firstSimple"].(map[string]interface{})
	if simple["text"] == "" {
		t.Error("expected a non-empty error message")
	}
}

// ---------------------------------------------------------------------------
// Alexa — full round-trip
// ---------------------------------------------------------------------------

func TestE2E_Alexa_FullRoundTrip(t *testing.T) {
	fakeOC, lastCmd := startFakeOpenClaw(t, "All systems operational")
	defer fakeOC.Close()

	bridge := startBridge(fakeOC.URL)
	defer bridge.Close()

	payload := `{
		"version":"1.0",
		"session":{"sessionId":"sess-abc"},
		"request":{
			"type":"IntentRequest",
			"requestId":"req-1",
			"intent":{"name":"CheckServerStatusIntent"}
		}
	}`
	resp, err := http.Post(bridge.URL+"/alexa", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /alexa: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if lastCmd.Command != "CheckServerStatusIntent" {
		t.Errorf("expected OpenClaw to receive CheckServerStatusIntent, got %q", lastCmd.Command)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	response := body["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] != "All systems operational" {
		t.Errorf("unexpected speech text: %q", speech["text"])
	}
}

func TestE2E_Alexa_WithSlots_ForwardedToOpenClaw(t *testing.T) {
	fakeOC, lastCmd := startFakeOpenClaw(t, "Restarted database")
	defer fakeOC.Close()

	bridge := startBridge(fakeOC.URL)
	defer bridge.Close()

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
	resp, err := http.Post(bridge.URL+"/alexa", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /alexa: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if lastCmd.Params["service"] != "database" {
		t.Errorf("expected slot service=database forwarded to OpenClaw, got %q", lastCmd.Params["service"])
	}
}

func TestE2E_Alexa_LaunchRequest_WelcomeMessage(t *testing.T) {
	bridge := startBridge("http://127.0.0.1:1") // openclaw unused for launch request
	defer bridge.Close()

	payload := `{"version":"1.0","request":{"type":"LaunchRequest"}}`
	resp, err := http.Post(bridge.URL+"/alexa", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /alexa: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	response := body["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] == "" {
		t.Error("expected a non-empty welcome message")
	}
}

func TestE2E_Alexa_OpenClawDown_ReturnsFriendlyMessage(t *testing.T) {
	bridge := startBridge("http://127.0.0.1:1")
	defer bridge.Close()

	payload := `{"version":"1.0","request":{"type":"IntentRequest","intent":{"name":"CheckServerStatusIntent"}}}`
	resp, err := http.Post(bridge.URL+"/alexa", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("POST /alexa: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	response := body["response"].(map[string]interface{})
	speech := response["outputSpeech"].(map[string]interface{})
	if speech["text"] == "" {
		t.Error("expected a non-empty error message")
	}
}
