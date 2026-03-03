package openclaw_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-zen-chu/homeclaw-bridge/internal/openclaw"
)

func TestSendCommand_Success(t *testing.T) {
	want := &openclaw.CommandResponse{
		Status:  "ok",
		Message: "Server is running",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/command" {
			t.Errorf("expected path /command, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var req openclaw.CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if req.Command != "CheckServerStatus" {
			t.Errorf("expected command CheckServerStatus, got %s", req.Command)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := openclaw.NewClient(srv.URL)
	got, err := client.SendCommand(context.Background(), openclaw.CommandRequest{
		Command: "CheckServerStatus",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != want.Status {
		t.Errorf("Status: got %q, want %q", got.Status, want.Status)
	}
	if got.Message != want.Message {
		t.Errorf("Message: got %q, want %q", got.Message, want.Message)
	}
}

func TestSendCommand_WithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openclaw.CommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if req.Params["target"] != "database" {
			t.Errorf("expected param target=database, got %q", req.Params["target"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&openclaw.CommandResponse{Status: "ok", Message: "done"})
	}))
	defer srv.Close()

	client := openclaw.NewClient(srv.URL)
	_, err := client.SendCommand(context.Background(), openclaw.CommandRequest{
		Command: "Restart",
		Params:  map[string]string{"target": "database"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendCommand_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := openclaw.NewClient(srv.URL)
	_, err := client.SendCommand(context.Background(), openclaw.CommandRequest{Command: "Anything"})
	if err == nil {
		t.Fatal("expected an error for non-200 response, got nil")
	}
}

func TestSendCommand_InvalidURL(t *testing.T) {
	client := openclaw.NewClient("http://127.0.0.1:1") // nothing listening here
	_, err := client.SendCommand(context.Background(), openclaw.CommandRequest{Command: "Anything"})
	if err == nil {
		t.Fatal("expected an error for unreachable URL, got nil")
	}
}

func TestSendCommand_InvalidResponseJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := openclaw.NewClient(srv.URL)
	_, err := client.SendCommand(context.Background(), openclaw.CommandRequest{Command: "Anything"})
	if err == nil {
		t.Fatal("expected an error for invalid JSON response, got nil")
	}
}
