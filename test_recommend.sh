#!/bin/bash
# 推薦系統測試 - 驗證 intent 分類 + 推薦結果
# 測試三種意圖：recommend、order（含 intent 欄位）、other

API_URL="http://localhost:3000/api/order"
PASS=0
FAIL=0
TOTAL=0

# ============================================================
# 推薦測試：驗證 intent=recommend + recommendations 非空
# ============================================================
test_recommend() {
    local test_name="$1"
    local input="$2"
    local min_recs="$3"  # 最少推薦數量

    TOTAL=$((TOTAL + 1))

    echo ""
    echo -e "\033[1;35m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
    echo -e "\033[1;35m[推薦] $test_name\033[0m"
    echo "輸入: $input"
    echo -e "\033[1;35m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$input\"}")

    intent=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('intent',''))" 2>/dev/null)
    rec_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('recommendations',[])))" 2>/dev/null)
    order_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('orders',[])))" 2>/dev/null)
    rec_items=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(f\"  {r['item']} ({r.get('category','')}) - {r.get('reason','')}\") for r in d.get('recommendations',[])]" 2>/dev/null)

    local test_passed=true

    # 檢查 intent
    if [ "$intent" = "recommend" ]; then
        echo -e "\033[0;32m✓ intent=recommend\033[0m"
    else
        echo -e "\033[0;31m✗ intent 錯誤: 預期 recommend, 實際 $intent\033[0m"
        test_passed=false
    fi

    # 檢查推薦數量
    if [ "$rec_count" -ge "$min_recs" ] 2>/dev/null; then
        echo -e "\033[0;32m✓ 推薦 $rec_count 個品項 (>= $min_recs)\033[0m"
    else
        echo -e "\033[0;31m✗ 推薦數量不足: 預期 >= $min_recs, 實際 $rec_count\033[0m"
        test_passed=false
    fi

    # 檢查不應有訂單
    if [ "$order_count" = "0" ]; then
        echo -e "\033[0;32m✓ 無訂單 (純推薦)\033[0m"
    else
        echo -e "\033[0;31m✗ 不應有訂單, 實際 $order_count 筆\033[0m"
        test_passed=false
    fi

    # 顯示推薦內容
    if [ -n "$rec_items" ]; then
        echo -e "\033[0;36m推薦品項:\033[0m"
        echo "$rec_items"
    fi

    if [ "$test_passed" = true ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo -e "\033[0;33mRAW: $(echo "$response" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),ensure_ascii=False,indent=2))" 2>/dev/null | head -20)\033[0m"
    fi
}

# ============================================================
# 點餐測試（帶 intent 欄位）：驗證 intent=order + orders 非空
# ============================================================
test_order_intent() {
    local test_name="$1"
    local input="$2"
    local expected_count="$3"
    shift 3
    local expected_items=("$@")

    TOTAL=$((TOTAL + 1))

    echo ""
    echo -e "\033[1;33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
    echo -e "\033[1;33m[點餐] $test_name\033[0m"
    echo "輸入: $input"
    echo -e "\033[1;33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$input\"}")

    intent=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('intent',''))" 2>/dev/null)
    actual_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('orders',[])))" 2>/dev/null)
    actual_items=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(o.get('item','')) for o in d.get('orders',[])]" 2>/dev/null)

    local test_passed=true

    # 檢查 intent
    if [ "$intent" = "order" ]; then
        echo -e "\033[0;32m✓ intent=order\033[0m"
    else
        echo -e "\033[0;31m✗ intent 錯誤: 預期 order, 實際 $intent\033[0m"
        test_passed=false
    fi

    # 檢查訂單數量
    if [ "$actual_count" = "$expected_count" ]; then
        echo -e "\033[0;32m✓ 訂單數量正確: $actual_count\033[0m"
    else
        echo -e "\033[0;31m✗ 訂單數量錯誤: 預期 $expected_count, 實際 $actual_count\033[0m"
        test_passed=false
    fi

    # 檢查品項
    for item in "${expected_items[@]}"; do
        if echo "$actual_items" | grep -q "$item"; then
            echo -e "\033[0;32m✓ 包含: $item\033[0m"
        else
            echo -e "\033[0;31m✗ 缺少: $item\033[0m"
            test_passed=false
        fi
    done

    if [ "$test_passed" = true ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo -e "\033[0;33mRAW: $(echo "$response" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),ensure_ascii=False,indent=2))" 2>/dev/null | head -20)\033[0m"
    fi
}

# ============================================================
# 閒聊測試：驗證 intent=other
# ============================================================
test_other() {
    local test_name="$1"
    local input="$2"

    TOTAL=$((TOTAL + 1))

    echo ""
    echo -e "\033[1;36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
    echo -e "\033[1;36m[閒聊] $test_name\033[0m"
    echo "輸入: $input"
    echo -e "\033[1;36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$input\"}")

    intent=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('intent',''))" 2>/dev/null)
    order_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('orders',[])))" 2>/dev/null)
    rec_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('recommendations',[])))" 2>/dev/null)
    note=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('note',''))" 2>/dev/null)

    local test_passed=true

    # 檢查 intent
    if [ "$intent" = "other" ]; then
        echo -e "\033[0;32m✓ intent=other\033[0m"
    else
        echo -e "\033[0;31m✗ intent 錯誤: 預期 other, 實際 $intent\033[0m"
        test_passed=false
    fi

    # 不應有訂單
    if [ "$order_count" = "0" ]; then
        echo -e "\033[0;32m✓ 無訂單\033[0m"
    else
        echo -e "\033[0;31m✗ 不應有訂單, 實際 $order_count 筆\033[0m"
        test_passed=false
    fi

    # 不應有推薦
    if [ "$rec_count" = "0" ]; then
        echo -e "\033[0;32m✓ 無推薦\033[0m"
    else
        echo -e "\033[0;31m✗ 不應有推薦, 實際 $rec_count 筆\033[0m"
        test_passed=false
    fi

    if [ -n "$note" ]; then
        echo -e "\033[0;36mNote: $note\033[0m"
    fi

    if [ "$test_passed" = true ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo -e "\033[0;33mRAW: $(echo "$response" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),ensure_ascii=False,indent=2))" 2>/dev/null | head -20)\033[0m"
    fi
}

# ============================================================
# 混合意圖測試：驗證 orders + recommendations 同時存在
# ============================================================
test_mixed() {
    local test_name="$1"
    local input="$2"
    local expected_order_count="$3"
    local expected_has_recs="$4"  # "yes" or "no"
    shift 4
    local expected_items=("$@")

    TOTAL=$((TOTAL + 1))

    echo ""
    echo -e "\033[1;34m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
    echo -e "\033[1;34m[混合] $test_name\033[0m"
    echo "輸入: $input"
    echo -e "\033[1;34m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$input\"}")

    intent=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('intent',''))" 2>/dev/null)
    order_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('orders',[])))" 2>/dev/null)
    rec_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('recommendations',[])))" 2>/dev/null)
    actual_items=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(o.get('item','')) for o in d.get('orders',[])]" 2>/dev/null)
    rec_items=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(f\"  {r['item']} - {r.get('reason','')}\") for r in d.get('recommendations',[])]" 2>/dev/null)

    local test_passed=true

    # 檢查 intent = order
    if [ "$intent" = "order" ]; then
        echo -e "\033[0;32m✓ intent=order\033[0m"
    else
        echo -e "\033[0;31m✗ intent 錯誤: 預期 order, 實際 $intent\033[0m"
        test_passed=false
    fi

    # 檢查訂單數量
    if [ "$order_count" = "$expected_order_count" ]; then
        echo -e "\033[0;32m✓ 訂單數量: $order_count\033[0m"
    else
        echo -e "\033[0;31m✗ 訂單數量: 預期 $expected_order_count, 實際 $order_count\033[0m"
        test_passed=false
    fi

    # 檢查推薦
    if [ "$expected_has_recs" = "yes" ]; then
        if [ "$rec_count" -gt 0 ] 2>/dev/null; then
            echo -e "\033[0;32m✓ 有推薦: $rec_count 個品項\033[0m"
        else
            echo -e "\033[0;31m✗ 應有推薦但沒有\033[0m"
            test_passed=false
        fi
    else
        if [ "$rec_count" = "0" ]; then
            echo -e "\033[0;32m✓ 無推薦 (已做決定)\033[0m"
        else
            echo -e "\033[0;31m✗ 不應有推薦 (已做決定), 實際 $rec_count 筆\033[0m"
            test_passed=false
        fi
    fi

    # 檢查品項
    for item in "${expected_items[@]}"; do
        if echo "$actual_items" | grep -q "$item"; then
            echo -e "\033[0;32m✓ 包含: $item\033[0m"
        else
            echo -e "\033[0;31m✗ 缺少: $item\033[0m"
            test_passed=false
        fi
    done

    # 顯示推薦內容
    if [ -n "$rec_items" ]; then
        echo -e "\033[0;36m推薦品項:\033[0m"
        echo "$rec_items"
    fi

    if [ "$test_passed" = true ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
        echo -e "\033[0;33mRAW: $(echo "$response" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin),ensure_ascii=False,indent=2))" 2>/dev/null | head -20)\033[0m"
    fi
}


echo "========================================"
echo "  推薦系統測試"
echo "========================================"

# ==========================================
echo -e "\n\033[0;34m【一：純推薦 - 直接推薦】\033[0m"
# ==========================================

test_recommend "有什麼推薦的" \
    "有什麼推薦的嗎" 1

test_recommend "推薦一下" \
    "推薦一下" 1

test_recommend "招牌是什麼" \
    "你們招牌是什麼" 1

test_recommend "第一次來" \
    "第一次來，推薦什麼" 1

test_recommend "不知道喝什麼" \
    "不知道要喝什麼" 1

test_recommend "幫我選" \
    "幫我選一杯吧" 1

# ==========================================
echo -e "\n\033[0;34m【二：純推薦 - 口味偏好】\033[0m"
# ==========================================

test_recommend "不太甜" \
    "有沒有不太甜的" 1

test_recommend "茶味重" \
    "我想喝茶味重一點的" 1

test_recommend "花香" \
    "有花香的推薦嗎" 1

test_recommend "清爽" \
    "清爽一點的有什麼" 1

test_recommend "不要奶" \
    "不要奶的" 1

# ==========================================
echo -e "\n\033[0;34m【三：純推薦 - 場景/溫度】\033[0m"
# ==========================================

test_recommend "夏天" \
    "夏天適合喝什麼" 1

test_recommend "想喝熱的" \
    "想喝熱的" 1

test_recommend "配甜點" \
    "想配甜點" 1

test_recommend "飯後" \
    "吃飯後適合喝什麼" 1

test_recommend "下午茶" \
    "下午茶適合什麼" 1

# ==========================================
echo -e "\n\033[0;34m【四：純推薦 - 分類查詢】\033[0m"
# ==========================================

test_recommend "純茶推薦" \
    "純茶有什麼推薦" 1

test_recommend "果茶推薦" \
    "果茶推薦" 1

test_recommend "奶蓋推薦" \
    "奶蓋推薦哪個" 1

test_recommend "鮮奶茶推薦" \
    "鮮奶茶推薦哪個" 1

# ==========================================
echo -e "\n\033[0;34m【五：點餐 - 驗證 intent=order】\033[0m"
# ==========================================

test_order_intent "基本點餐" \
    "三窨十五茉鮮奶半糖少冰" 1 "三窨十五茉"

test_order_intent "多品項" \
    "琥珀鐵觀音鮮奶半糖；桂花輕烏龍純茶去冰" 2 "琥珀鐵觀音" "桂花輕烏龍"

test_order_intent "加料點餐" \
    "金杯蜜香紅鮮奶加紅烏龍茶凍" 1 "金杯蜜香紅"

# ==========================================
echo -e "\n\033[0;34m【六：閒聊 - 驗證 intent=other】\033[0m"
# ==========================================

test_other "問Wi-Fi" \
    "有Wi-Fi嗎"

test_other "問廁所" \
    "廁所在哪"

test_other "問營業時間" \
    "你們營業到幾點"

test_other "問非菜單品項" \
    "你們有賣咖啡嗎"

# ==========================================
echo -e "\n\033[0;34m【七：混合意圖 - 先點再問推薦】\033[0m"
# ==========================================

test_mixed "點餐+推薦" \
    "三窨十五茉鮮奶半糖少冰；再推薦一個配甜點的" 1 "yes" "三窨十五茉"

test_mixed "點餐+問好喝" \
    "桂花輕烏龍純茶去冰無糖；還有什麼好喝的" 1 "yes" "桂花輕烏龍"

# ==========================================
echo -e "\n\033[0;34m【八：混合意圖 - 先推薦再點（推薦清空）】\033[0m"
# ==========================================

test_mixed "推薦後決定" \
    "推薦一下；好那桂花輕烏龍鮮奶半糖少冰" 1 "no" "桂花輕烏龍"

test_mixed "不知道→決定" \
    "不知道喝什麼；好那三窨十五茉純茶去冰無糖" 1 "no" "三窨十五茉"

# ==========================================
echo -e "\n\033[0;34m【九：混合意圖 - 點+推薦+點（推薦清空）】\033[0m"
# ==========================================

test_mixed "點+推薦+點" \
    "三窨十五茉鮮奶半糖；推薦果茶；那加一杯葡柚十五茉" 2 "no" "三窨十五茉" "葡柚十五茉"

# ==========================================

echo ""
echo "========================================"
echo "  測試結果總結"
echo "========================================"
echo -e "\033[0;32m通過: $PASS\033[0m"
echo -e "\033[0;31m失敗: $FAIL\033[0m"
echo "總計: $TOTAL"

if [ $TOTAL -gt 0 ]; then
    echo "通過率: $((PASS * 100 / TOTAL))%"
fi

if [ $FAIL -gt 0 ]; then
    echo -e "\n\033[1;33m⚠ 有 $FAIL 個測試失敗\033[0m"
fi
