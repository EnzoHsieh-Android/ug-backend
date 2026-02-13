# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API backend for a tea shop ordering system. Supports two modes:
1. **Legacy**: Direct text → order parsing (POST /api/order)
2. **Session**: Multi-segment voice recording → STT → order parsing with state management

**Tech Stack:** Go 1.23, chi/v5 router, go-pinyin for fuzzy matching

## Common Commands

```bash
# Run the server (requires LLM model on port 8080 + DashScope API key)
STT_API_KEY=<dashscope-key> go run main.go

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
- `STT_URL` - Qwen3-ASR-Flash API base URL (default: `https://dashscope-intl.aliyuncs.com`)
- `STT_API_KEY` - DashScope API key (required)
- `STT_MODEL` - ASR model name (default: `qwen3-asr-flash`)
- `STT_TIMEOUT` - STT request timeout (default: `60s`)
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
    ├── stt.go          # Qwen3-ASR-Flash STT client (DashScope API)
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
  → ParseIntentResponse() - parse JSON, validate against menu, fillDefaults
  → Return APIResponse with orders/recommendations/warnings
```

## Request Flow (Session)

```
CreateSession → [AddSegment (audio → STT)] × N → Finalize → [UpdateOrders] → Confirm → Complete

Finalize 內部流程：
  合併所有 segments (用 ";" 分隔) → NormalizeSTT → LLM Complete
  → ParseIntentResponse (validate + fillDefaults)
  → 過濾已送出推薦 (SentRecommendations 去重)
  → 合併 manual overrides → 回傳結果
```

## STT (service/stt.go)

Uses Qwen3-ASR-Flash via DashScope API (Singapore international endpoint).
Audio is base64 encoded and sent as JSON with menu context in system message for domain-specific recognition.

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

### fillDefaults (預設值填入)
模型輸出 null 表示「顧客未指定」，後端 `fillDefaults()` 負責填入預設值：
- **Category**: pure_tea 優先 → rich_milk_tea → 第一個匹配的分類
- **Size**: 有尺寸的品項預設 M，單一尺寸品項設為 nil
- **Sugar**: 預設「正常糖」
- **Ice**: 預設「正常冰」

### Intent 推導
模型不再輸出 intent，由 `ParseIntentResponse()` 從陣列內容推導：
- orders 非空 → "order"
- recommendations 非空（無 orders）→ "recommend"
- 兩者皆空 → "other"

### 推薦去重 (SentRecommendations)
Session 維護 `SentRecommendations map[string]bool`，多輪 finalize 時過濾已送出的推薦品項，避免重複推薦。Reset 時清空。

## Key Design Decisions

- **Pointer fields** (`*string`) in Order struct indicate optional customizations
- **NotOnMenu** flag marks items not found in menu instead of rejecting
- **Session state machine** - enforces valid state transitions (created → recording → pending_order → confirmed → completed)
- **LLM-dependent** - requires Qwen3-8B fine-tuned model
- **Null output strategy** - model outputs null for unspecified fields, backend fills defaults
- **Recommendations limited to 1** - system prompt specifies 「僅推薦 1 項」
- **Qwen3-ASR-Flash STT** - uses DashScope API with menu context in system message for domain-specific recognition

## Model Training

LLM fine-tuning 工作在 `/Users/xiechengru/ug-finetune/`，訓練前後的分析與優化流程遵照 `finetune-optimization` skill（詳見 WORKFLOW.md）。

目前模型：Qwen3-8B-UG-order-v10（Round 30, iter 800, 96% eval — 48/50, diversity 37/37 100%）
上一版：Qwen3-8B-UG-order-v9（Round 29, iter 700, 96% — 48/50）

### Round 30 變更
- 新增奶蓋/茶凍消歧訓練資料（cloud_velvet vs addon 紅烏龍茶凍）
- 後端 ice 別名正規化（"熱"/"熱的" → "熱飲"）
- 後端 inferCategoryFromText：從 user text 推斷多分類品項 category
- 修復 normalizer fuzzy pinyin 重疊窗口 bug（exact-match 位置保護）
- 訓練資料：1305 train + 313 valid = 1618 樣本
- 最佳 checkpoint：iter 800（核心功能 100%，跨分類 6/7）

### Round 29 變更
- 重平衡 fresh_milk_tea:rich_milk_tea 比例（2.85:1 → 1.60:1）
- 新增同品項跨分類訓練資料（鮮奶茶 vs 奶茶 vs 純茶 vs 奶蓋）
- 修正 system prompt 品項名錯誤（茶花烏龍 → 茶花紅烏龍）
- 後端新增 ColdOnly 防護（fruit_tea 不可選熱飲）
