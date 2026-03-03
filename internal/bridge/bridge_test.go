package bridge

import (
	"errors"
	"testing"
)

// mockHandler is a test double for DeviceHandler.
type mockHandler struct {
	result Result
	err    error
}

func (m *mockHandler) Execute(_ Command) (Result, error) {
	return m.result, m.err
}

func TestBridge_Execute(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(b *Bridge)
		cmd        Command
		wantResult Result
		wantErr    bool
	}{
		{
			name:  "device not found returns failure result",
			setup: func(_ *Bridge) {},
			cmd: Command{
				DeviceID: "unknown-device",
				Action:   "turn-on",
			},
			wantResult: Result{
				DeviceID: "unknown-device",
				Success:  false,
				Message:  "device not found: unknown-device",
			},
			wantErr: false,
		},
		{
			name: "registered handler executes successfully",
			setup: func(b *Bridge) {
				b.RegisterHandler("device-1", &mockHandler{
					result: Result{
						DeviceID: "device-1",
						Success:  true,
						Message:  "command executed",
					},
				})
			},
			cmd: Command{
				DeviceID: "device-1",
				Action:   "turn-on",
			},
			wantResult: Result{
				DeviceID: "device-1",
				Success:  true,
				Message:  "command executed",
			},
			wantErr: false,
		},
		{
			name: "handler error is propagated",
			setup: func(b *Bridge) {
				b.RegisterHandler("device-2", &mockHandler{
					err: errors.New("device communication error"),
				})
			},
			cmd: Command{
				DeviceID: "device-2",
				Action:   "turn-on",
			},
			wantResult: Result{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			tt.setup(b)

			got, err := b.Execute(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.DeviceID != tt.wantResult.DeviceID ||
					got.Success != tt.wantResult.Success ||
					got.Message != tt.wantResult.Message {
					t.Errorf("Execute() = %+v, want %+v", got, tt.wantResult)
				}
			}
		})
	}
}
