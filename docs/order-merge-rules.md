# 訂單合併規則（Order Merge Rules）

## 訂單來源

| 來源 | 觸發方式 | 範例 |
|------|----------|------|
| **語音** | Finalize（LLM 解析） | 「一杯桂花輕烏龍半糖少冰」 |
| **推薦採納** | 前端把推薦加入購物車（PATCH） | LLM 推薦茶花紅烏龍 → 使用者點「加入」 |
| **手動新增** | 前端直接新增品項（PATCH） | 使用者在 UI 手動選品項加入 |
| **手動修改** | 前端修改既有訂單規格（PATCH） | 改甜度、冰量、分類、數量、加料 |

推薦採納視同手動新增。

---

## 場景矩陣

### 基本流程

| # | 場景 | 流程 | 預期結果 |
|---|------|------|----------|
| 1 | 單輪語音點餐 | 語音 → Finalize → Confirm | 正常出單 |
| 2 | 語音 + 手動改規格 | Finalize → 改半糖→微糖 → Confirm | 微糖出單 |
| 3 | 語音 + 手動刪品項 | Finalize → 刪掉某品項 → Confirm | 該品項不出現 |
| 4 | 語音 + 手動加品項 | Finalize → 手動加一杯新飲料 → Confirm | 新品項出現 |
| 5 | 語音 + 採納推薦 | Finalize（含推薦）→ 加入推薦品項 → Confirm | 推薦品項視同手動新增 |

### 多輪語音（Re-finalize）

| # | 場景 | 流程 | 預期結果 |
|---|------|------|----------|
| 6 | 純多輪語音 | Finalize → 加語音「再一杯茶花烏龍」→ Re-finalize | 新品項出現，舊品項保留 |
| 7 | 手動改規格 + 再語音 | Finalize → 改半糖 → 加語音 → Re-finalize | 半糖保留（LLM 沒改該欄位） |
| 8 | 手動刪品項 + 再語音（語音未提到） | Finalize → 刪 A → 加語音 → Re-finalize | A 不出現（deleted flag） |
| 9 | 手動加品項 + 再語音 | Finalize → 手動加 B → 加語音 → Re-finalize | B 保留（不受 LLM 結果影響） |
| 10 | 手動改分類 + 再語音 | Finalize → 改純茶→鮮奶茶 → 加語音 → Re-finalize | 分類改動保留 |
| 11 | 推薦採納 + 再語音 | Finalize → 採納推薦 C → 加語音 → Re-finalize | C 保留（視同手動新增） |
| 12 | 語音覆蓋手動改動 | Finalize → 改半糖 → 語音說「改微糖」→ Re-finalize | LLM 回微糖 → 語音贏（LLM 主動改了） |
| 13 | 手動刪 + 語音重新提到 | Finalize → 刪 A → 語音說「加一杯 A」→ Re-finalize | A 重新出現（新 ID），語音覆蓋刪除 |

### 重複品項

| # | 場景 | 流程 | 預期結果 |
|---|------|------|----------|
| 14 | 語音點同品項不同規格 | 「兩杯桂花，一杯半糖一杯無糖」→ Finalize | 兩筆獨立訂單，各自 ID |
| 15 | 手動加同品項不同規格 | Finalize → 手動加同品項但不同甜度 | 兩筆獨立，不互相干擾 |
| 16 | 重複品項 + 手動改 + 再語音 | 場景 14 → 改其中一筆 → Re-finalize | 各自 override 獨立 |
| 17 | 語音品項 + 手動加同品項 | Finalize 得 A(半糖) → 手動加 A(無糖) → Re-finalize | 語音 A 走 merge，手動 A 獨立保留 |

### 特殊場景

| # | 場景 | 預期結果 |
|---|------|----------|
| 18 | Reset | 全部清空（segments、orders、overrides、ID 計數器） |
| 19 | 推薦去重 | 同品項不重複推薦（SentRecommendations 追蹤） |
| 20 | 語音只聊天（無點餐意圖）| intent="other"，既有訂單不受影響 |

---

## 衝突判定規則

### Re-finalize 時的欄位合併

對每個欄位（category / sugar / ice / size / addon / qty / note）：

```
LLM 值 ≠ 上次 Base 值？
  → 是：LLM 主動改了 → 用 LLM 值，清除手動 override（語音贏）
  → 否：LLM 沒改 → 檢查手動 override
            有 → 用手動值（手動贏）
            無 → 用 LLM 值
```

### 品項層級規則

| 情況 | 處理 |
|------|------|
| LLM 回傳 + 無 override | 正常使用 LLM 結果 |
| LLM 回傳 + 有 override | 欄位層級 merge（見上方規則） |
| LLM 回傳 + 已 deleted | 不參與匹配，視為新訂單（新 ID）→ 語音覆蓋刪除 |
| LLM 未回傳 + manually added | 保留手動新增的品項 |
| LLM 未回傳 + 非手動新增 | 品項被語音移除，不出現 |

---

## 匹配演算法（Re-finalize）

```
1. 從 s.Orders（有序）收集可匹配 overrides
   條件：非 deleted、非 manually added

2. 每筆 LLM order 依序嘗試匹配：
   a. 精確匹配：item + base category 都相同 → 取第一個可用的
   b. 退回匹配：僅 item 相同（處理 category 被 override 的情況）
   c. 匹配成功 → merge fields + 保留原 ID，從可匹配列表移除
   d. 無匹配 → assign 新 ID，建立新 override

3. 加入所有 manually added + 非 deleted 的品項（含推薦採納）
```

---

## 技術設計

### Order ID
- 每筆 Order 由後端 assign 唯一 ID（遞增整數字串 "1", "2", "3"...）
- LLM 不輸出 ID
- 前端從 response 取得 ID，PATCH 時回傳

### Overrides Map
- Key：order ID（非 item name）
- 追蹤欄位：category / sugar / ice / size / addon / qty / note
- 標記：deleted / manuallyAdded

### 預設值
- 由後端 fillDefaults() 處理（category / sugar / ice / size）
- 在 merge 之前執行，merge 看到的是已填預設值的 LLM 結果
