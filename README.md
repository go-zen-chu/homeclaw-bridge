# homeclaw-bridge

A local-network HTTP bridge that forwards smart speaker commands (Google Home,
Alexa) to an [OpenClaw](https://github.com/go-zen-chu) server running on the
same home network.  **All traffic stays on the local network — no audio or
request data ever leaves your home.**

---

## Architecture

```
┌──────────────────┐        POST /google-home        ┌────────────────────┐
│  Google Home     │ ──────────────────────────────▶ │                    │
│  (local device)  │                                  │  homeclaw-bridge   │
└──────────────────┘        POST /alexa               │  (this server)     │
                                                      │                    │
┌──────────────────┐ ──────────────────────────────▶ │  :8080             │
│  Alexa           │                                  └────────┬───────────┘
│  (local device)  │                                           │ POST /command
└──────────────────┘                                           ▼
                                                      ┌────────────────────┐
                                                      │  OpenClaw server   │
                                                      │  (local network)   │
                                                      └────────────────────┘
```

## Endpoints

| Method | Path           | Description                                              |
|--------|----------------|----------------------------------------------------------|
| POST   | `/google-home` | Google Actions SDK webhook (local fulfillment)           |
| POST   | `/alexa`       | Alexa Custom Skill webhook                               |
| GET    | `/health`      | Liveness probe — returns `{"status":"ok"}`               |

## Configuration

All configuration is done via environment variables:

| Variable       | Default                    | Description                         |
|----------------|----------------------------|-------------------------------------|
| `LISTEN_ADDR`  | `:8080`                    | Address the bridge listens on       |
| `OPENCLAW_URL` | `http://localhost:9090`    | Base URL of the local OpenClaw API  |

## Getting started

### Prerequisites

* Go 1.24 or later
* OpenClaw server running on the local network

### Build & run

```bash
go build -o homeclaw-bridge ./cmd/homeclaw-bridge
OPENCLAW_URL=http://192.168.1.10:9090 ./homeclaw-bridge
```

### Run tests

```bash
go test ./...
```

## Request formats

### Google Home (Actions SDK)

```json
POST /google-home
{
  "handler": { "name": "CheckServerStatus" },
  "intent":  { "name": "actions.intent.MAIN" }
}
```

Response:

```json
{
  "prompt": {
    "override": false,
    "firstSimple": {
      "speech": "Server is running",
      "text":   "Server is running"
    }
  }
}
```

### Alexa Custom Skill

```json
POST /alexa
{
  "version": "1.0",
  "session": { "sessionId": "..." },
  "request": {
    "type":      "IntentRequest",
    "requestId": "...",
    "intent": {
      "name":  "CheckServerStatusIntent",
      "slots": {}
    }
  }
}
```

Response:

```json
{
  "version": "1.0",
  "response": {
    "outputSpeech": {
      "type": "PlainText",
      "text": "Server is running"
    },
    "shouldEndSession": true
  }
}
```

## OpenClaw API contract

The bridge sends `POST /command` to the configured `OPENCLAW_URL`:

```json
{ "command": "CheckServerStatus", "params": {} }
```

Expected response:

```json
{ "status": "ok", "message": "Server is running" }
```

## License

See [LICENSE](LICENSE).
