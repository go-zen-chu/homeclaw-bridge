package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zen-chu/homeclaw-bridge/internal/bridge"
	"github.com/go-zen-chu/homeclaw-bridge/internal/server"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	b := bridge.New()
	s := server.New(b)
	return httptest.NewServer(s)
}

func TestE2E_HealthCheck(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz") //nolint:noctx
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestE2E_HealthCheck_MethodNotAllowed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/healthz", "application/json", nil) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestE2E_Commands_DeviceNotFound(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	cmd := bridge.Command{
		DeviceID: "unknown-device",
		Action:   "turn-on",
	}
	body, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}

	resp, err := http.Post(ts.URL+"/commands", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /commands: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result bridge.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for unknown device")
	}
	if result.DeviceID != "unknown-device" {
		t.Errorf("DeviceID = %q, want %q", result.DeviceID, "unknown-device")
	}
}

func TestE2E_Commands_InvalidBody(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/commands", "application/json", bytes.NewReader([]byte("not-json"))) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /commands: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestE2E_Commands_MethodNotAllowed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/commands", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /commands: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
