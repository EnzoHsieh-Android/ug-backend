#!/bin/bash
# 多樣性測試 v2 - 使用菜單上的實際品項
# 菜單品項：三窨十五茉、桂花輕烏龍、茶花紅烏龍、朱槿普洱紅、米香玉露菁
#          茶花烏龍、琥珀鐵觀音、金杯蜜香紅、青檸桂蜜、蜜桃紅烏龍、葡柚十五茉

API_URL="http://localhost:3000/api/order"
PASS=0
FAIL=0

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

test_order() {
    local desc="$1"
    local text="$2"
    local expected_items="$3"
    local expected_count="$4"
    local check_not_on_menu="${5:-no}"  # "no" = should NOT have not_on_menu items

    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}測試: $desc${NC}"
    echo -e "輸入: $text"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$text\"}")

    echo "回應:"
    echo "$response" | jq '.'

    local test_passed=true

    # Check order count
    actual_count=$(echo "$response" | jq '.orders | length')
    if [ "$actual_count" == "$expected_count" ]; then
        echo -e "${GREEN}✓ 訂單數量正確: $actual_count${NC}"
    else
        echo -e "${RED}✗ 訂單數量錯誤: 預期 $expected_count, 實際 $actual_count${NC}"
        test_passed=false
    fi

    # Check expected items
    if [ -n "$expected_items" ]; then
        for item in $expected_items; do
            if echo "$response" | jq -e ".orders[] | select(.item == \"$item\")" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ 包含: $item${NC}"
            else
                echo -e "${RED}✗ 缺少: $item${NC}"
                test_passed=false
            fi
        done
    fi

    # Check for not_on_menu
    not_on_menu=$(echo "$response" | jq '[.orders[] | select(.not_on_menu == true)] | length')
    if [ "$check_not_on_menu" == "no" ] && [ "$not_on_menu" -gt 0 ]; then
        echo -e "${RED}✗ 有 $not_on_menu 個品項被標記為不在菜單上${NC}"
        test_passed=false
    elif [ "$not_on_menu" -gt 0 ]; then
        echo -e "${YELLOW}⚠ 有 $not_on_menu 個品項不在菜單上 (預期)${NC}"
    fi

    if [ "$test_passed" = true ]; then
        ((PASS++))
    else
        ((FAIL++))
    fi
}

echo "========================================"
echo "  多樣性測試 v2 - 使用實際菜單品項"
echo "========================================"

# ==================== 1. 基本新增測試 ====================
echo -e "\n${BLUE}【類別一：基本新增測試】${NC}"

test_order "單品 - 純茶" \
    "一杯三窨十五茉" \
    "三窨十五茉" \
    "1"

test_order "單品 - 鮮奶茶" \
    "桂花輕烏龍鮮奶" \
    "桂花輕烏龍" \
    "1"

test_order "單品 - 果茶" \
    "青檸桂蜜" \
    "青檸桂蜜" \
    "1"

test_order "多品項 - 三種不同類別" \
    "我要三窨十五茉、金杯蜜香紅鮮奶、還有青檸桂蜜" \
    "三窨十五茉 金杯蜜香紅 青檸桂蜜" \
    "3"

test_order "多品項 - 同品項多杯" \
    "三杯桂花輕烏龍" \
    "桂花輕烏龍" \
    "1"

# ==================== 2. 縮寫識別測試 ====================
echo -e "\n${BLUE}【類別二：縮寫識別測試】${NC}"

test_order "縮寫 - 茉莉 → 三窨十五茉" \
    "一杯茉莉" \
    "三窨十五茉" \
    "1"

test_order "縮寫 - 桂花 → 桂花輕烏龍" \
    "桂花半糖少冰" \
    "桂花輕烏龍" \
    "1"

test_order "縮寫 - 鐵觀音 → 琥珀鐵觀音" \
    "鐵觀音微糖去冰" \
    "琥珀鐵觀音" \
    "1"

test_order "縮寫 - 普洱 → 朱槿普洱紅" \
    "普洱熱的" \
    "朱槿普洱紅" \
    "1"

test_order "縮寫 - 金杯 → 金杯蜜香紅" \
    "金杯無糖" \
    "金杯蜜香紅" \
    "1"

test_order "縮寫 - 蜜桃 → 蜜桃紅烏龍" \
    "蜜桃果茶" \
    "蜜桃紅烏龍" \
    "1"

# ==================== 3. 客製化選項測試 ====================
echo -e "\n${BLUE}【類別三：客製化選項測試】${NC}"

test_order "選項 - 甜度" \
    "三窨十五茉半糖" \
    "三窨十五茉" \
    "1"

test_order "選項 - 冰量" \
    "金杯蜜香紅去冰" \
    "金杯蜜香紅" \
    "1"

test_order "選項 - 尺寸" \
    "大杯桂花輕烏龍奶茶" \
    "桂花輕烏龍" \
    "1"

test_order "選項 - 加料" \
    "琥珀鐵觀音加紅烏龍茶凍" \
    "琥珀鐵觀音" \
    "1"

test_order "選項 - 雙加料" \
    "朱槿普洱紅加紅烏龍茶凍和燕麥粒" \
    "朱槿普洱紅" \
    "1"

test_order "選項 - 全客製" \
    "大杯三窨十五茉微糖去冰加燕麥粒" \
    "三窨十五茉" \
    "1"

test_order "選項 - 熱飲" \
    "米香玉露菁熱的無糖" \
    "米香玉露菁" \
    "1"

# ==================== 4. 修改操作測試 ====================
echo -e "\n${BLUE}【類別四：修改操作測試】${NC}"

test_order "修改 - 改品項" \
    "我要三窨十五茉，不對，改成桂花輕烏龍" \
    "桂花輕烏龍" \
    "1"

test_order "修改 - 改甜度" \
    "金杯蜜香紅半糖，啊不對，改無糖" \
    "金杯蜜香紅" \
    "1"

test_order "修改 - 多次猶豫" \
    "茉莉，不對普洱，還是桂花好了" \
    "桂花輕烏龍" \
    "1"

# ==================== 5. 刪除操作測試 ====================
echo -e "\n${BLUE}【類別五：刪除操作測試】${NC}"

test_order "刪除 - 取消一項" \
    "三窨十五茉和金杯蜜香紅，金杯不要了" \
    "三窨十五茉" \
    "1"

test_order "刪除 - 取消加料" \
    "琥珀鐵觀音加茶凍，茶凍不加了" \
    "琥珀鐵觀音" \
    "1"

# ==================== 6. 組合操作測試 ====================
echo -e "\n${BLUE}【類別六：組合操作測試】${NC}"

test_order "組合 - 新增後修改" \
    "三窨十五茉和金杯蜜香紅，茉莉改半糖" \
    "三窨十五茉 金杯蜜香紅" \
    "2"

test_order "組合 - 多品項各有客製" \
    "三窨十五茉半糖、金杯蜜香紅去冰、琥珀鐵觀音熱的" \
    "三窨十五茉 金杯蜜香紅 琥珀鐵觀音" \
    "3"

test_order "組合 - 複雜場景" \
    "兩杯茉莉半糖和一杯桂花去冰，茉莉改微糖" \
    "三窨十五茉 桂花輕烏龍" \
    "2"

# ==================== 7. 口語化表達測試 ====================
echo -e "\n${BLUE}【類別七：口語化表達】${NC}"

test_order "口語 - 猶豫表達" \
    "嗯...我想要...茉莉吧，對，就茉莉" \
    "三窨十五茉" \
    "1"

test_order "口語 - 朋友代點" \
    "我要三窨十五茉，我朋友要金杯蜜香紅" \
    "三窨十五茉 金杯蜜香紅" \
    "2"

test_order "口語 - 選擇困難" \
    "茉莉還是桂花呢...算了兩個都要" \
    "三窨十五茉 桂花輕烏龍" \
    "2"

# ==================== 8. 類別識別測試 ====================
echo -e "\n${BLUE}【類別八：類別識別測試】${NC}"

test_order "類別 - 純茶" \
    "三窨十五茉純茶" \
    "三窨十五茉" \
    "1"

test_order "類別 - 鮮奶" \
    "金杯蜜香紅鮮奶" \
    "金杯蜜香紅" \
    "1"

test_order "類別 - 奶茶 (UG奶茶)" \
    "桂花輕烏龍奶茶" \
    "桂花輕烏龍" \
    "1"

test_order "類別 - 奶蓋" \
    "茶花紅烏龍奶蓋" \
    "茶花紅烏龍" \
    "1"

# ==================== 9. 邊界情況測試 ====================
echo -e "\n${BLUE}【類別九：邊界情況】${NC}"

test_order "邊界 - 重複品項不同選項" \
    "三窨十五茉半糖、三窨十五茉無糖" \
    "三窨十五茉" \
    "2"

test_order "邊界 - 超長訂單" \
    "三窨十五茉、桂花輕烏龍、琥珀鐵觀音、金杯蜜香紅、青檸桂蜜" \
    "三窨十五茉 桂花輕烏龍 琥珀鐵觀音 金杯蜜香紅 青檸桂蜜" \
    "5"

# ==================== 10. 非菜單品項測試 ====================
echo -e "\n${BLUE}【類別十：非菜單品項 (預期標記 not_on_menu)】${NC}"

test_order "非菜單 - 珍珠奶茶" \
    "一杯珍珠奶茶" \
    "珍珠奶茶" \
    "1" \
    "yes"

test_order "非菜單 - 紅茶拿鐵" \
    "紅茶拿鐵" \
    "紅茶拿鐵" \
    "1" \
    "yes"

# ==================== 總結 ====================
echo ""
echo "========================================"
echo "  測試結果總結"
echo "========================================"
echo -e "${GREEN}通過: $PASS${NC}"
echo -e "${RED}失敗: $FAIL${NC}"
TOTAL=$((PASS + FAIL))
RATE=$((PASS * 100 / TOTAL))
echo "總計: $TOTAL"
echo "通過率: ${RATE}%"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ 所有測試通過！${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ 有 $FAIL 個測試失敗，請檢查模型訓練資料${NC}"
    exit 1
fi
