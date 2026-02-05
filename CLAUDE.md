# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API backend for a tea shop ordering system. Supports two modes:
1. **Legacy**: Direct text → order parsing (POST /api/order)
2. **Session**: Multi-segment voice recording → STT → order parsing with state management

**Tech Stack:** Go 1.23, chi/v5 router, go-pinyin for fuzzy matching

## Common Commands

```bash
# Run the server (requires LLM model on port 8080 + Whisper STT)
WHISPER_URL=https://api.openai.com WHISPER_API_KEY=<key> go run main.go

# Build
go build -o ug-backend .
CGO_ENABLED=0 go build -o ug-backend .  # For Docker/Alpine

# Run all tests
go test ./...

# Run specific test file
go test ./service/normalizer_test.go ./service/normalizer.go

# Run diversity tests (requires server running)
./test_diversity_v2.sh    # Standard tests (37 cases)
./test_diversity_v3.sh    # Variation tests (40 cases)

# Docker
docker build -t ug-backend .
docker-compose up
```

## Environment Variables

- `PORT` - Server port (default: `3000`)
- `MODEL_URL` - LLM API endpoint (default: `http://localhost:8080`)
- `WHISPER_URL` - Whisper STT API base URL (default: `http://localhost:9000`)
- `WHISPER_API_KEY` - Whisper API key (default: empty)
- `WHISPER_MODEL` - Whisper model name (default: `whisper-1`)
- `WHISPER_TIMEOUT` - STT request timeout (default: `30s`)
- `SESSION_TTL` - Session expiry time (default: `30m`)
- `MAX_SEGMENTS_PER_SESSION` - Max voice segments per session (default: `20`)
- `MAX_AUDIO_SIZE_MB` - Max audio upload size (default: `25`)

## Architecture

```
main.go                 # Entry point, chi router setup
├── config/config.go    # Environment variable loading
├── handler/
│   ├── health.go       # GET /api/health
│   ├── menu.go         # GET /api/menu
│   ├── order.go        # POST /api/order (legacy)
│   └── session.go      # /api/session/* (session-based flow)
├── middleware/cors.go   # CORS middleware
├── model/
│   ├── menu.go         # Menu item structs
│   ├── order.go        # Order struct and API response types
│   └── session.go      # Session state machine and structs
└── service/
    ├── llm.go          # LLM client (calls /v1/chat/completions)
    ├── menu.go         # Menu data and lookup
    ├── normalizer.go   # STT typo correction (exact + fuzzy Pinyin)
    ├── session.go      # Session lifecycle management
    ├── stt.go          # Whisper STT client (OpenAI compatible)
    └── validator.go    # Order validation against menu
```

## API Endpoints

### Legacy
```
POST /api/order {"text": "..."} → orders JSON
```

### Session Flow
```
POST   /api/session              # Create session
GET    /api/session/{id}         # Get session state
DELETE /api/session/{id}         # Delete session
POST   /api/session/{id}/segment   # Upload audio segment → STT
POST   /api/session/{id}/finalize  # Merge segments → LLM → orders
PATCH  /api/session/{id}/orders    # Manual order edits
POST   /api/session/{id}/confirm   # Confirm orders
POST   /api/session/{id}/back      # Go back to editing
POST   /api/session/{id}/complete  # Complete (payment done)
POST   /api/session/{id}/reset     # Reset to recording
POST   /api/session/{id}/cancel    # Cancel session
```

## Request Flow (Legacy)

```
POST /api/order {"text": "..."}
  → NormalizeSTT() - two-tier typo correction
  → LLMClient.Complete() - send to model
  → ParseAndValidate() - parse JSON, validate against menu
  → Return APIResponse with orders/warnings
```

## Request Flow (Session)

```
CreateSession → [AddSegment (audio → STT)] × N → Finalize → [UpdateOrders] → Confirm → Complete
```

## STT Normalization System (service/normalizer.go)

Two-layer approach for handling speech-to-text errors:

1. **Exact Replacement Table** (~60 known typos): Fast O(n) string replacement for common STT errors (e.g., "三因" → "三窨")

2. **Pinyin Fuzzy Matching**: For unknown typos, uses sliding window with 80% similarity threshold on toneless Pinyin

## Menu & Validation (service/validator.go)

12 menu items, all in Traditional Chinese. Valid options:
- Sweetness: 正常糖, 少糖, 半糖, 微糖, 無糖
- Ice: 正常冰, 少冰, 微冰, 去冰, 熱飲
- Size: M, L
- Add-ons: 紅烏龍茶凍, 燕麥粒

## Key Design Decisions

- **Pointer fields** (`*string`) in Order struct indicate optional customizations
- **NotOnMenu** flag marks items not found in menu instead of rejecting
- **Session state machine** - enforces valid state transitions (created → recording → pending_order → confirmed → completed)
- **LLM-dependent** - requires Qwen3-8B fine-tuned model (currently v4, 97% accuracy)
- **OpenAI-compatible STT** - uses Whisper API with menu prompt for domain-specific recognition

## Model Training

LLM fine-tuning 工作在 `/Users/xiechengru/ug-finetune/`，訓練前後的分析與優化流程遵照 `finetune-optimization` skill（詳見 WORKFLOW.md）。

目前模型：Qwen3-8B-UG-order-v4（Round 22, iter 1000, 97% accuracy）
