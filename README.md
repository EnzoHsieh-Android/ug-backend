# UG Backend

No.315 Unique Green 茶飲店語音點餐後端系統。

## 功能

- **語音點餐**：多段錄音 → Qwen3-ASR-Flash 語音辨識 → 拼音校正 → LLM 解析訂單
- **智慧推薦**：根據顧客偏好推薦品項（每次限 1 項，Session 內去重）
- **手動編輯**：前端可修改訂單，re-finalize 時智慧合併（手動修改不被覆蓋）
- **Order ID 系統**：每筆訂單唯一 ID，支援同品項多筆、跨分類修改
- **拼音校正**：兩層 STT 糾錯（精確替換表 + 拼音模糊匹配）
- **預設值填入**：模型輸出 null，後端自動填入甜度/冰量/尺寸/分類預設值

## 技術棧

- Go 1.23 + chi/v5 路由
- Qwen3-ASR-Flash（DashScope API）語音辨識
- Qwen3-8B Fine-tuned 模型（MLX 推理）
- go-pinyin 拼音模糊匹配

## 快速開始

```bash
# 需要：LLM server 運行在 :8080 + DashScope API Key
STT_API_KEY=<your-dashscope-key> go run main.go
```

伺服器預設啟動在 `:3000`。

## 環境變數

| 變數 | 預設值 | 說明 |
|------|--------|------|
| `PORT` | `3000` | 伺服器埠號 |
| `MODEL_URL` | `http://localhost:8080` | LLM API 端點 |
| `STT_URL` | `https://dashscope-intl.aliyuncs.com` | DashScope API |
| `STT_API_KEY` | （必填） | DashScope API Key |
| `STT_MODEL` | `qwen3-asr-flash` | ASR 模型 |
| `STT_TIMEOUT` | `60s` | STT 請求逾時 |
| `SESSION_TTL` | `30m` | Session 過期時間 |
| `MAX_SEGMENTS_PER_SESSION` | `20` | 每 Session 最多音檔段落 |
| `MAX_AUDIO_SIZE_MB` | `25` | 音檔大小上限 |

## API 端點

### 健康檢查 & 菜單

| 方法 | 路徑 | 說明 |
|------|------|------|
| `GET` | `/api/health` | 健康檢查 |
| `GET` | `/api/menu` | 取得完整菜單 |

### Legacy（單次文字點餐）

| 方法 | 路徑 | 說明 |
|------|------|------|
| `POST` | `/api/order` | 文字 → 訂單 JSON |

### Session（語音點餐流程）

| 方法 | 路徑 | 說明 |
|------|------|------|
| `POST` | `/api/session` | 建立 Session |
| `GET` | `/api/session/{id}` | 取得 Session 狀態 |
| `DELETE` | `/api/session/{id}` | 刪除 Session |
| `POST` | `/api/session/{id}/segment` | 上傳音檔段落（multipart） |
| `POST` | `/api/session/{id}/finalize` | 合併段落 → 分析訂單 |
| `PATCH` | `/api/session/{id}/orders` | 手動修改訂單 |
| `POST` | `/api/session/{id}/reset` | 重置回錄音狀態 |
| `POST` | `/api/session/{id}/confirm` | 確認訂單 |
| `POST` | `/api/session/{id}/back` | 返回編輯 |
| `POST` | `/api/session/{id}/complete` | 完成付款 |
| `POST` | `/api/session/{id}/cancel` | 取消 Session |

## Session 狀態機

```
POST /session → active ── finalize ──→ pending ── confirm ──→ checkout ── complete ──→ completed
                  ↑                      │  ↑                    │
                  └────── reset ─────────┘  └──── back ──────────┘
```

## 核心流程

```
錄音 → 上傳音檔 → STT 轉文字 → 拼音校正 → 回傳校正後文字
                                              ↓ (finalize)
                                    合併所有段落 → LLM 解析
                                              ↓
                                    驗證 + 填預設值 → 合併手動修改 → 回傳訂單
```

## 專案結構

```
main.go                 # 進入點，chi 路由設定
├── config/config.go    # 環境變數載入
├── handler/
│   ├── health.go       # 健康檢查
│   ├── menu.go         # 菜單 API
│   ├── order.go        # Legacy 單次點餐
│   └── session.go      # Session 語音點餐流程
├── middleware/cors.go   # CORS 中介層
├── model/
│   ├── menu.go         # 菜單資料結構
│   ├── order.go        # 訂單、推薦、Intent 結構
│   └── session.go      # Session 狀態機、Override 結構
└── service/
    ├── llm.go          # LLM 客戶端（/v1/chat/completions）
    ├── menu.go         # 菜單資料與模糊匹配
    ├── normalizer.go   # STT 拼音校正（精確 + 模糊）
    ├── session.go      # Session 生命週期、Order ID、Override 合併
    ├── stt.go          # Qwen3-ASR-Flash 客戶端（DashScope）
    └── validator.go    # 訂單驗證、預設值填入、推薦驗證
```

## 前端串接

詳見 [FRONTEND_GUIDE.md](FRONTEND_GUIDE.md)。

## 建置 & 部署

```bash
# 編譯
go build -o ug-backend .

# Docker
CGO_ENABLED=0 go build -o ug-backend .
docker build -t ug-backend .
docker-compose up
```
