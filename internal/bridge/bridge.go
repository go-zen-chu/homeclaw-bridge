package bridge

import "fmt"

// Command represents a device command to be executed.
type Command struct {
	DeviceID string                 `json:"device_id"`
	Action   string                 `json:"action"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

// Result represents the result of a command execution.
type Result struct {
	DeviceID string `json:"device_id"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
}

// DeviceHandler is the interface for handling device commands.
type DeviceHandler interface {
	Execute(cmd Command) (Result, error)
}

// Bridge routes commands to registered device handlers.
type Bridge struct {
	handlers map[string]DeviceHandler
}

// New creates a new Bridge.
func New() *Bridge {
	return &Bridge{
		handlers: make(map[string]DeviceHandler),
	}
}

// RegisterHandler registers a handler for the given device ID.
func (b *Bridge) RegisterHandler(deviceID string, handler DeviceHandler) {
	b.handlers[deviceID] = handler
}

// Execute routes the command to the appropriate handler and returns the result.
func (b *Bridge) Execute(cmd Command) (Result, error) {
	handler, ok := b.handlers[cmd.DeviceID]
	if !ok {
		return Result{
			DeviceID: cmd.DeviceID,
			Success:  false,
			Message:  fmt.Sprintf("device not found: %s", cmd.DeviceID),
		}, nil
	}
	return handler.Execute(cmd)
}
