#!/bin/bash
# 多樣性測試 - 包含新增、刪除、修改組合的複雜對話

API_URL="http://localhost:3000/api/order"
PASS=0
FAIL=0

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_order() {
    local desc="$1"
    local text="$2"
    local expected_items="$3"
    local expected_count="$4"

    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}測試: $desc${NC}"
    echo -e "輸入: $text"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$text\"}")

    echo "回應:"
    echo "$response" | jq '.'

    # Check order count
    actual_count=$(echo "$response" | jq '.orders | length')

    if [ "$actual_count" == "$expected_count" ]; then
        echo -e "${GREEN}✓ 訂單數量正確: $actual_count${NC}"
        ((PASS++))
    else
        echo -e "${RED}✗ 訂單數量錯誤: 預期 $expected_count, 實際 $actual_count${NC}"
        ((FAIL++))
    fi

    # Check expected items
    if [ -n "$expected_items" ]; then
        for item in $expected_items; do
            if echo "$response" | jq -e ".orders[] | select(.item == \"$item\")" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ 包含: $item${NC}"
            else
                echo -e "${RED}✗ 缺少: $item${NC}"
            fi
        done
    fi

    # Check for not_on_menu
    not_on_menu=$(echo "$response" | jq '[.orders[] | select(.not_on_menu == true)] | length')
    if [ "$not_on_menu" -gt 0 ]; then
        echo -e "${YELLOW}⚠ 有 $not_on_menu 個品項不在菜單上${NC}"
    fi

    echo ""
}

echo "========================================"
echo "  多樣性測試 - 複雜對話場景"
echo "========================================"

# ==================== 1. 新增類測試 ====================
echo -e "\n${YELLOW}【類別一：新增組合測試】${NC}"

test_order "多品項新增 - 三種飲料" \
    "我要一杯珍珠奶茶、一杯三窨十五茉、還有一杯桂花輕烏龍" \
    "珍珠奶茶 三窨十五茉 桂花輕烏龍" \
    "3"

test_order "多品項新增 - 同品項多杯" \
    "我要三杯珍珠奶茶" \
    "珍珠奶茶" \
    "1"

test_order "多品項新增 - 混合數量" \
    "兩杯金杯蜜香紅和三杯冬瓜檸檬" \
    "金杯蜜香紅 冬瓜檸檬" \
    "2"

# ==================== 2. 修改類測試 ====================
echo -e "\n${YELLOW}【類別二：修改操作測試】${NC}"

test_order "修改 - 改品項" \
    "我要珍珠奶茶，不對，改成紅茶拿鐵" \
    "紅茶拿鐵" \
    "1"

test_order "修改 - 改甜度冰量" \
    "我要珍珠奶茶半糖少冰，啊不對，改微糖去冰" \
    "珍珠奶茶" \
    "1"

test_order "修改 - 改尺寸" \
    "給我一杯中杯珍珠奶茶，改大杯" \
    "珍珠奶茶" \
    "1"

test_order "修改 - 多次修改" \
    "我要綠茶，不對紅茶，還是茉莉綠茶好了" \
    "茉莉綠茶" \
    "1"

# ==================== 3. 刪除類測試 ====================
echo -e "\n${YELLOW}【類別三：刪除操作測試】${NC}"

test_order "刪除 - 取消一項" \
    "我要珍珠奶茶和紅茶拿鐵，紅茶拿鐵不要了" \
    "珍珠奶茶" \
    "1"

test_order "刪除 - 數量減少" \
    "我要三杯珍珠奶茶，改成兩杯就好" \
    "珍珠奶茶" \
    "1"

test_order "刪除 - 取消加料" \
    "珍珠奶茶加紅烏龍茶凍，茶凍不要了" \
    "珍珠奶茶" \
    "1"

# ==================== 4. 組合操作測試 ====================
echo -e "\n${YELLOW}【類別四：新增+修改+刪除組合】${NC}"

test_order "組合 - 新增後修改" \
    "我要珍珠奶茶和紅茶拿鐵，珍珠奶茶改半糖" \
    "珍珠奶茶 紅茶拿鐵" \
    "2"

test_order "組合 - 新增後刪除再新增" \
    "我要珍珠奶茶，不要了，給我紅茶拿鐵" \
    "紅茶拿鐵" \
    "1"

test_order "組合 - 複雜修改刪除" \
    "三杯珍珠奶茶、兩杯紅茶拿鐵，珍珠奶茶改兩杯，紅茶拿鐵不要了" \
    "珍珠奶茶" \
    "1"

test_order "組合 - 多品項各有修改" \
    "珍珠奶茶半糖、紅茶拿鐵少冰、三窨十五茉熱的，珍珠奶茶改微糖" \
    "珍珠奶茶 紅茶拿鐵 三窨十五茉" \
    "3"

# ==================== 5. 口語化表達測試 ====================
echo -e "\n${YELLOW}【類別五：口語化表達】${NC}"

test_order "口語 - 猶豫表達" \
    "嗯...我想要...珍珠奶茶吧，對，就珍珠奶茶" \
    "珍珠奶茶" \
    "1"

test_order "口語 - 推薦詢問" \
    "有什麼推薦的嗎？那就來一杯三窨十五茉好了" \
    "三窨十五茉" \
    "1"

test_order "口語 - 朋友代點" \
    "我要珍珠奶茶，我朋友要紅茶拿鐵" \
    "珍珠奶茶 紅茶拿鐵" \
    "2"

test_order "口語 - 選擇困難" \
    "珍珠奶茶還是紅茶拿鐵呢...算了兩個都要" \
    "珍珠奶茶 紅茶拿鐵" \
    "2"

# ==================== 6. 縮寫與別名測試 ====================
echo -e "\n${YELLOW}【類別六：縮寫與別名】${NC}"

test_order "縮寫 - 茉莉" \
    "一杯茉莉" \
    "三窨十五茉" \
    "1"

test_order "縮寫 - 桂花" \
    "桂花一杯" \
    "桂花輕烏龍" \
    "1"

test_order "縮寫 - 鐵觀音" \
    "鐵觀音少糖去冰" \
    "琥珀鐵觀音" \
    "1"

test_order "縮寫 - 普洱" \
    "普洱熱的" \
    "朱槿普洱紅" \
    "1"

# ==================== 7. 特殊選項組合 ====================
echo -e "\n${YELLOW}【類別七：特殊選項組合】${NC}"

test_order "選項 - 全客製" \
    "珍珠奶茶大杯微糖去冰加紅烏龍茶凍" \
    "珍珠奶茶" \
    "1"

test_order "選項 - 雙加料" \
    "珍珠奶茶加紅烏龍茶凍和燕麥粒" \
    "珍珠奶茶" \
    "1"

test_order "選項 - 熱飲" \
    "三窨十五茉熱的無糖" \
    "三窨十五茉" \
    "1"

# ==================== 8. 邊界情況測試 ====================
echo -e "\n${YELLOW}【類別八：邊界情況】${NC}"

test_order "邊界 - 重複品項" \
    "珍珠奶茶半糖、珍珠奶茶全糖" \
    "珍珠奶茶" \
    "2"

test_order "邊界 - 矛盾指令" \
    "珍珠奶茶熱的少冰" \
    "珍珠奶茶" \
    "1"

test_order "邊界 - 超長訂單" \
    "珍珠奶茶、紅茶拿鐵、三窨十五茉、桂花輕烏龍、冬瓜檸檬" \
    "珍珠奶茶 紅茶拿鐵 三窨十五茉 桂花輕烏龍 冬瓜檸檬" \
    "5"

# ==================== 總結 ====================
echo ""
echo "========================================"
echo "  測試結果總結"
echo "========================================"
echo -e "${GREEN}通過: $PASS${NC}"
echo -e "${RED}失敗: $FAIL${NC}"
TOTAL=$((PASS + FAIL))
echo "總計: $TOTAL"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ 所有測試通過！${NC}"
    exit 0
else
    echo -e "${RED}✗ 有 $FAIL 個測試失敗${NC}"
    exit 1
fi
