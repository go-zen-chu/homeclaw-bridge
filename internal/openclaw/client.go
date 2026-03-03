// Package openclaw provides a client for communicating with the OpenClaw server
// running on the local network. All communication stays within the local network.
package openclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CommandRequest represents a command sent to the OpenClaw server.
type CommandRequest struct {
	Command string            `json:"command"`
	Params  map[string]string `json:"params,omitempty"`
}

// CommandResponse represents the response from the OpenClaw server.
type CommandResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Caller is the interface for sending commands to OpenClaw.
type Caller interface {
	SendCommand(ctx context.Context, cmd CommandRequest) (*CommandResponse, error)
}

// Client is an HTTP client for the OpenClaw server.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new OpenClaw client targeting the given base URL
// (e.g. "http://192.168.1.10:9090").
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendCommand sends a command to OpenClaw and returns its response.
func (c *Client) SendCommand(ctx context.Context, cmd CommandRequest) (*CommandResponse, error) {
	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshaling command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/command", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to OpenClaw: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenClaw returned status %d", resp.StatusCode)
	}

	var result CommandResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}
