# UG Backend 前端串接指南

> **API 測試位置：** `http://192.168.0.122:3000`
>
> 以下所有端點路徑前面加上此 base URL 即可呼叫，例如 `http://192.168.0.122:3000/api/health`

## 目錄

1. [系統架構總覽](#1-系統架構總覽)
2. [API 端點一覽](#2-api-端點一覽)
3. [資料型別定義](#3-資料型別定義)
4. [Session 狀態機](#4-session-狀態機)
5. [核心流程：語音點餐](#5-核心流程語音點餐)
6. [手動編輯訂單（PATCH /orders）](#6-手動編輯訂單patch-orders)
7. [Order ID 系統與衝突處理](#7-order-id-系統與衝突處理)
8. [菜單結構](#8-菜單結構)
9. [串接注意事項](#9-串接注意事項)
10. [錯誤處理](#10-錯誤處理)
11. [完整流程範例](#11-完整流程範例)

---

## 1. 系統架構總覽

```
┌─────────┐    audio     ┌──────────────┐   base64    ┌─────────────┐
│ Frontend │ ──────────→ │  UG Backend  │ ─────────→ │ DashScope   │
│  (Web)   │             │  (Go :3000)  │            │ ASR (STT)   │
└─────────┘             └──────────────┘            └─────────────┘
     ↑                        │
     │   JSON response        │ normalized text
     │                        ↓
     │                  ┌──────────────┐
     └───────────────── │  LLM Server  │
                        │  (MLX :8080) │
                        └──────────────┘
```

**後端職責：**
- 管理 Session 生命週期（建立 → 錄音 → 分析 → 確認 → 結帳 → 完成）
- 接收音檔 → 呼叫 STT 轉文字 → 校正拼音錯誤 → 呼叫 LLM 解析訂單
- 追蹤手動修改（field-level override），re-finalize 時智慧合併
- 每筆訂單分配唯一 ID，支援同品項多筆、分類修改等場景

---

## 2. API 端點一覽

| 方法 | 路徑 | 狀態要求 | 說明 |
|------|------|----------|------|
| `GET` | `/api/health` | — | 健康檢查 |
| `GET` | `/api/menu` | — | 取得完整菜單 |
| `POST` | `/api/order` | — | Legacy 單次文字點餐 |
| `POST` | `/api/session` | — | 建立新 Session |
| `GET` | `/api/session/{id}` | 任何 | 取得 Session 狀態 |
| `DELETE` | `/api/session/{id}` | 任何 | 刪除 Session |
| `POST` | `/api/session/{id}/segment` | active / pending | 上傳音檔段落 |
| `POST` | `/api/session/{id}/finalize` | active / pending | 合併段落 → 分析訂單 |
| `PATCH` | `/api/session/{id}/orders` | pending | 手動修改訂單 |
| `POST` | `/api/session/{id}/reset` | pending | 重置回錄音狀態 |
| `POST` | `/api/session/{id}/confirm` | pending | 確認訂單 → 結帳 |
| `POST` | `/api/session/{id}/back` | checkout | 返回編輯 |
| `POST` | `/api/session/{id}/complete` | checkout | 完成付款 |
| `POST` | `/api/session/{id}/cancel` | 非 completed | 取消 Session |

---

## 3. 資料型別定義

### Order（訂單項目）

```typescript
interface Order {
  id?: string;           // 後端分配的唯一 ID（"1", "2", "3"...）
  item: string;          // 品項名稱，如 "三窨十五茉"
  category?: string;     // 分類 ID（見下方菜單結構）
  size: string | null;   // "M" | "L" | null（單一尺寸品項為 null）
  sugar: string | null;  // "正常糖" | "少糖" | "半糖" | "微糖" | "無糖"
  ice: string | null;    // "正常冰" | "少冰" | "微冰" | "去冰" | "熱飲"
  addon: string[];       // ["紅烏龍茶凍", "燕麥粒"]（空陣列 = 無加料）
  qty: number;           // 數量，最小 1
  note?: string;         // 備註
  not_on_menu?: boolean; // true = 品項不在菜單上
}
```

### Recommendation（推薦品項）

```typescript
interface Recommendation {
  item: string;      // 品項名稱
  category: string;  // 分類 ID
  reason: string;    // 推薦理由
}
```

### SessionResponse（Session API 統一回應）

```typescript
interface SessionResponse {
  session_id?: string;
  status?: "active" | "pending" | "checkout" | "completed" | "cancelled";
  intent?: "order" | "recommend" | "other";
  merged_text?: string;          // 合併後的語音文字
  orders?: Order[];
  recommendations?: Recommendation[];
  note?: string;                 // 模型回覆（閒聊、提示等）
  warnings?: string[];           // 驗證警告
  error?: string;                // 錯誤訊息
}
```

### SegmentResponse（上傳音檔回應）

```typescript
interface SegmentResponse {
  segment_index: number;   // 此段落的 index（0-based）
  text: string;            // 拼音校正後的轉錄文字（可直接顯示）
  total_segments: number;  // 目前總段落數
  error?: string;
}
```

---

## 4. Session 狀態機

```
                    ┌─────────────────────────────────┐
                    │           cancel（任何時候）       │
                    │              ↓                   │
    POST /session → active ──finalize──→ pending ──confirm──→ checkout ──complete──→ completed
                      ↑                    │  ↑                  │
                      │                    │  │                  │
                      └────── reset ───────┘  └──── back ────────┘
                                               (PATCH /orders 可編輯)
```

**關鍵規則：**
- `active`：可上傳音檔（segment）、可 finalize
- `pending`：可 PATCH 編輯訂單、可再次 finalize（追加語音）、可 confirm / reset
- `checkout`：僅可 back 或 complete
- `completed` / `cancelled`：終態，不可操作

---

## 5. 核心流程：語音點餐

### Step 1：建立 Session

```http
POST /api/session
Content-Type: application/json
```

```json
// Response
{ "session_id": "abc-123-def", "status": "active" }
```

### Step 2：上傳音檔

```http
POST /api/session/{id}/segment
Content-Type: multipart/form-data

audio: (binary file)
```

支援格式：`.wav`, `.mp3`, `.m4a`, `.mp4`, `.webm`, `.ogg`, `.opus`, `.flac`
大小上限：預設 25 MB

```json
// Response
{ "segment_index": 0, "text": "我要一杯三窨十五茉，半糖去冰", "total_segments": 1 }
```

**`text` 已經過拼音校正**（例如 ASR 聽成「三因十五茉」會自動修正為「三窨十五茉」），前端可直接顯示。

可多次上傳，每次 finalize 時所有段落以 `;` 合併。

### 即時顯示語音文字

每次上傳 segment 後，前端用回傳的 `text` 逐行顯示：

```
第 1 段: "我要一杯三窨十五茉，半糖去冰"
第 2 段: "再加一杯桂花輕烏龍，熱的"
```

多輪語音逐行累加顯示即可，文字已校正、可直接呈現給使用者。

### Step 3：Finalize（分析訂單）

```http
POST /api/session/{id}/finalize
```

```json
// Response
{
  "session_id": "abc-123-def",
  "status": "pending",
  "intent": "order",
  "merged_text": "我要一杯三窨十五茉，半糖去冰",
  "orders": [
    {
      "id": "1",
      "item": "三窨十五茉",
      "category": "pure_tea",
      "size": null,
      "sugar": "半糖",
      "ice": "去冰",
      "addon": [],
      "qty": 1
    }
  ],
  "recommendations": [
    { "item": "桂花輕烏龍", "category": "pure_tea", "reason": "同為花香系列" }
  ],
  "warnings": []
}
```

**重要：** Finalize 可在 `pending` 狀態再次呼叫（使用者追加語音後 re-finalize），此時會觸發 override merge 邏輯。

### 多輪語音：Re-finalize 前同步修改

如果使用者在第一輪 finalize 後手動改了訂單，又錄了第二輪語音，re-finalize 之前必須先把修改同步到後端，否則手動修改會被 LLM 新結果覆蓋。

```
第一輪: segments → finalize → 顯示訂單
使用者手動修改訂單（本地狀態）
第二輪: 錄新語音 → 上傳 segments
  → 按「分析」前，比對本地訂單有無修改
  → 有差異 → 先 PATCH /orders，再 POST /finalize
  → 沒差異 → 直接 POST /finalize
```

### Step 4：確認 → 結帳 → 完成

使用者按「送出訂單」時，前端應先比對本地訂單與 finalize 回傳的訂單是否有差異：

```
使用者按「送出」
  → 比對本地訂單 vs finalize 回來的訂單
  → 有差異 → 先 PATCH /orders 同步修改，再 POST /confirm
  → 沒差異 → 直接 POST /confirm（省一次請求）
```

```http
PATCH /api/session/{id}/orders   → 有修改時才呼叫
POST /api/session/{id}/confirm   → status: "checkout"
POST /api/session/{id}/complete  → status: "completed"
```

---

## 6. 手動編輯訂單（PATCH /orders）

### 基本用法

```http
PATCH /api/session/{id}/orders
Content-Type: application/json
```

前端將**整個訂單陣列**送出（不是 diff），後端自動比對差異。

### 修改現有訂單

保留原本的 `id`，修改需要的欄位：

```json
{
  "orders": [
    { "id": "1", "item": "三窨十五茉", "category": "pure_tea",
      "sugar": "微糖", "ice": "去冰", "size": null, "addon": [], "qty": 1 }
  ]
}
```

### 新增品項

不帶 `id`（或 `id` 為空字串），後端會自動分配新 ID 並標記為「手動新增」：

```json
{
  "orders": [
    { "id": "1", "item": "三窨十五茉", ... },
    { "item": "琥珀鐵觀音", "category": "rich_milk_tea",
      "sugar": "少糖", "ice": "少冰", "size": "M", "addon": [], "qty": 1 }
  ]
}
```

回應中該品項會帶有新 `id`（如 `"2"`）。

### 刪除品項

從陣列中移除即可。後端標記為 deleted，re-finalize 時不會復活：

```json
{
  "orders": [
    // 原本 id "1" 和 "2"，現在只留 "1" → "2" 被標記 deleted
    { "id": "1", "item": "三窨十五茉", ... }
  ]
}
```

### 修改分類

直接改 `category` 欄位，後端會追蹤此修改：

```json
{
  "orders": [
    { "id": "1", "item": "三窨十五茉", "category": "fresh_milk_tea", ... }
  ]
}
```

---

## 7. Order ID 系統與衝突處理

### 為什麼需要 Order ID

| 場景 | 舊系統（item name 當 key） | 新系統（Order ID） |
|------|---------------------------|-------------------|
| 改分類後 re-finalize | 分類被覆蓋 | 正確保留手動分類 |
| 同品項不同分類 | 無法區分 | 各有獨立 ID |
| 同品項多筆（一杯半糖一杯無糖） | 衝突崩潰 | 各有獨立 ID |

### 衝突判定規則

Re-finalize 時，每個欄位的勝負判定：

```
LLM 新值 ≠ 上次 LLM 值 (Base)  →  語音贏，清除手動 override
LLM 新值 == 上次 LLM 值 (Base)  →  檢查手動 override
  ├── 有手動 override → 手動贏
  └── 無手動 override → 用 LLM 值
```

追蹤的欄位：`category`, `sugar`, `ice`, `size`, `addon`, `qty`, `note`

### 匹配演算法

LLM 不輸出 ID，後端用匹配將 LLM 結果對應到現有訂單：

1. **精確匹配**：`item` + `base category` 都相同 → 配對
2. **退回匹配**：僅 `item` 相同 → 配對（處理分類被手動改過的情況）
3. **無匹配** → 視為新訂單，分配新 ID

### 特殊規則

- **手動刪除的品項**：不參與匹配。若語音重新提到同品項 → 新 ID
- **手動新增的品項**：不受 LLM 影響，始終保留
- **推薦採納**：前端將推薦加入訂單時，視同手動新增（不帶 ID）

---

## 8. 菜單結構

### GET /api/menu 回應結構

```typescript
interface Menu {
  categories: Category[];
  options: {
    sugar: string[];   // ["正常糖", "少糖", "半糖", "微糖", "無糖"]
    ice: string[];     // ["正常冰", "少冰", "微冰", "去冰", "熱飲"]
    size: string[];    // ["M", "L"]
    addon: Addon[];    // [{ name, price }]
  };
}

interface Category {
  id: string;       // "pure_tea" | "fresh_milk_tea" | "rich_milk_tea" | "fruit_tea" | "cloud_velvet"
  name: string;     // 顯示名稱
  items: MenuItem[];
}

interface MenuItem {
  name: string;       // 品項名稱
  price_m?: number;   // M 杯價格（無則為單一尺寸）
  price_l: number;    // L 杯價格
  has_size: boolean;  // true = 有 M/L 選擇
  cold_only: boolean; // true = 僅供冷飲
}
```

### 分類 ID 對照

| ID | 名稱 | 尺寸 | 備註 |
|----|------|------|------|
| `pure_tea` | 研選純茶 | 單一 (L) | |
| `fresh_milk_tea` | 牧場直達 | M / L | |
| `rich_milk_tea` | UG奶茶 | M / L | |
| `fruit_tea` | 果香四溢 | 單一 | 僅冷飲 |
| `cloud_velvet` | 雲頂奶蓋 | 單一 | |

### 同品項跨分類

部分品項出現在多個分類中（如「三窨十五茉」可以是純茶或鮮奶茶），`category` 欄位決定實際分類。

---

## 9. 串接注意事項

### CORS

後端允許 `*` 來源，支援 `GET, POST, PATCH, DELETE, OPTIONS`，Header 允許 `Content-Type`。

### 請求格式

- JSON 端點：`Content-Type: application/json`
- 音檔上傳：`Content-Type: multipart/form-data`，欄位名為 `audio`

### Order ID 處理規則

1. **Finalize 回來的每筆 order 都帶 `id`** — 前端必須保存並在 PATCH 時回傳
2. **PATCH 時保留 `id`** — 修改現有訂單必須帶原 `id`
3. **新增品項不帶 `id`**（或空字串）— 後端自動分配
4. **刪除品項**：從陣列中移除即可
5. **回應中所有 order 都會有 `id`** — 包含新分配的

### Null 與預設值

- `size`, `sugar`, `ice` 可能為 `null`
- Finalize 後後端已填入預設值（size=M, sugar=正常糖, ice=正常冰）
- 前端不需自行處理預設值邏輯
- 單一尺寸品項的 `size` 會是 `null`（不是 "L"），前端顯示時可忽略尺寸選擇

### Intent 判讀

| intent | 含義 | 前端行為建議 |
|--------|------|-------------|
| `"order"` | 有訂單 | 顯示訂單列表 |
| `"recommend"` | 純推薦（無訂單） | 顯示推薦卡片 |
| `"other"` | 閒聊 / 無法辨識 | 顯示 `note` 內容 |

### 推薦處理

- 每次 finalize 只回傳**尚未送過**的推薦（後端去重）
- 最多 1 項推薦
- 使用者採納推薦 → 前端用 PATCH 將推薦加入 orders（不帶 `id`）
- Reset 後推薦記錄清空

### 語音文字顯示

- `POST /segment` 回傳的 `text` **已經過拼音校正**，可直接顯示
- 前端累加每段文字，多輪用換行呈現（不需等 finalize）
- Finalize 後的 `merged_text` 是所有段落用 `;` 合併的完整文字

### 多輪語音追加

使用者可在 `pending` 狀態繼續上傳音檔並 re-finalize：
1. 上傳新音檔 → `POST /segment`（此時 status 仍為 pending，允許操作）
2. 前端即時顯示該段校正後文字（換行追加）
3. Re-finalize → `POST /finalize`
4. 後端合併**所有**段落（含之前的），重新呼叫 LLM
5. Merge 邏輯保留手動修改，智慧合併新結果

### Session 過期

- 預設 TTL：30 分鐘（從最後操作算起）
- 過期後 GET 會回 404
- 前端應處理 session 消失的情況

---

## 10. 錯誤處理

### HTTP Status Codes

| Code | 情境 |
|------|------|
| 200 | 成功 |
| 204 | DELETE 成功（無 body） |
| 400 | 請求錯誤、狀態不允許、無段落等 |
| 404 | Session 不存在 |
| 502 | STT 服務失敗 |

### 錯誤回應格式

所有錯誤都在同一結構中，帶 `error` 欄位：

```json
{ "error": "cannot finalize: session status is checkout" }
```

### 常見錯誤訊息

| error | 說明 |
|-------|------|
| `"session not found: {id}"` | Session 不存在或已過期 |
| `"cannot add segment: session status is {s}"` | 狀態不允許上傳 |
| `"cannot finalize: session status is {s}"` | 狀態不允許 finalize |
| `"cannot update orders: session status is {s}"` | 非 pending 不能編輯 |
| `"no segments to finalize"` | 沒有音檔段落 |
| `"maximum segments (20) reached"` | 段落數已達上限 |
| `"LLM error: ..."` | LLM 服務異常 |
| `"transcription failed: ..."` | STT 轉錄失敗 |

---

## 11. 完整流程範例

### 場景：使用者語音點兩杯茶，手動改一杯的甜度，再語音追加一杯

```
1. POST /api/session
   → { session_id: "s1", status: "active" }

2. POST /api/session/s1/segment  (音檔: "我要兩杯三窨十五茉，一杯半糖一杯無糖")
   → { segment_index: 0, text: "我要兩杯三窨十五茉，一杯半糖一杯無糖" }

3. POST /api/session/s1/finalize
   → {
       status: "pending",
       intent: "order",
       orders: [
         { id: "1", item: "三窨十五茉", sugar: "半糖", ice: "正常冰", ... },
         { id: "2", item: "三窨十五茉", sugar: "無糖", ice: "正常冰", ... }
       ]
     }

4. PATCH /api/session/s1/orders  (使用者把第一杯改成微糖)
   → request: {
       orders: [
         { id: "1", item: "三窨十五茉", sugar: "微糖", ice: "正常冰", ... },
         { id: "2", item: "三窨十五茉", sugar: "無糖", ice: "正常冰", ... }
       ]
     }

5. POST /api/session/s1/segment  (音檔: "再一杯桂花輕烏龍，熱的")
   → { segment_index: 1, text: "再一杯桂花輕烏龍，熱的" }

6. POST /api/session/s1/finalize  (re-finalize，合併兩段語音)
   → {
       orders: [
         { id: "1", item: "三窨十五茉", sugar: "微糖", ... },  ← 保留手動改的微糖
         { id: "2", item: "三窨十五茉", sugar: "無糖", ... },  ← 保留原設定
         { id: "3", item: "桂花輕烏龍", sugar: "正常糖", ice: "熱飲", ... }  ← 新品項
       ]
     }

7. POST /api/session/s1/confirm
   → { status: "checkout" }

8. POST /api/session/s1/complete
   → { status: "completed" }
```

### 場景：使用者採納推薦

```
1. Finalize 回應含推薦:
   recommendations: [{ item: "桂花輕烏龍", category: "pure_tea", reason: "..." }]

2. 使用者按「加入」→ 前端 PATCH:
   {
     orders: [
       ...existing orders...,
       { "item": "桂花輕烏龍", "category": "pure_tea",
         "sugar": "正常糖", "ice": "正常冰", "size": null,
         "addon": [], "qty": 1 }          ← 不帶 id
     ]
   }

3. 回應中新品項自動帶 id:
   { id: "3", item: "桂花輕烏龍", ... }   ← 後端分配
```

### 場景：手動改分類後 re-finalize

```
1. Finalize: { id: "1", item: "三窨十五茉", category: "pure_tea", size: null, ... }

2. PATCH 改分類: { id: "1", item: "三窨十五茉", category: "fresh_milk_tea", size: "M", ... }

3. Re-finalize → LLM 仍回 category: "pure_tea"
   → 合併結果: category: "fresh_milk_tea" (手動贏), size: "M" (手動贏)
   ← 因為 LLM 的 category 沒變（== base），手動 override 保留
```
