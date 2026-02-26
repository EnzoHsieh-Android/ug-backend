#!/bin/bash
# 泛化測試 - 全新 phrasing，不重複訓練資料
# 目的：測試模型對未見過的自然語言表達的泛化能力

API_URL="http://localhost:3000/api/order"
PASS=0
FAIL=0
TOTAL=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Enhanced test function with field-level checks
# Usage: test_order "desc" "text" "expected_count" "check_spec"
# check_spec format: "item:field=value|item:field=value|..."
# Fields: item, category, ice, sugar, size, addon, qty, not_on_menu
# Special: intent=order|recommend|other, rec_count=N
test_order() {
    local desc="$1"
    local text="$2"
    local expected_count="$3"
    local check_spec="$4"

    ((TOTAL++))

    echo -e "\n${CYAN}─── 測試 #$TOTAL: $desc ───${NC}"
    echo -e "輸入: ${YELLOW}$text${NC}"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$text\"}")

    local test_passed=true

    # Check order count
    actual_count=$(echo "$response" | jq '.orders | length')
    if [ "$actual_count" == "$expected_count" ]; then
        echo -e "${GREEN}  ✓ 訂單數: $actual_count${NC}"
    else
        echo -e "${RED}  ✗ 訂單數: 預期 $expected_count, 實際 $actual_count${NC}"
        test_passed=false
    fi

    # Parse check_spec
    if [ -n "$check_spec" ]; then
        IFS='|' read -ra CHECKS <<< "$check_spec"
        for check in "${CHECKS[@]}"; do
            # Handle intent check
            if [[ "$check" == intent=* ]]; then
                expected_intent="${check#intent=}"
                actual_intent=$(echo "$response" | jq -r '.intent // empty')
                if [ "$actual_intent" == "$expected_intent" ]; then
                    echo -e "${GREEN}  ✓ intent=$actual_intent${NC}"
                else
                    echo -e "${RED}  ✗ intent: 預期 $expected_intent, 實際 $actual_intent${NC}"
                    test_passed=false
                fi
                continue
            fi

            # Handle rec_count check
            if [[ "$check" == rec_count=* ]]; then
                expected_rec="${check#rec_count=}"
                actual_rec=$(echo "$response" | jq '.recommendations | length')
                if [ "$actual_rec" -ge "$expected_rec" ]; then
                    echo -e "${GREEN}  ✓ 推薦數: $actual_rec (>=$expected_rec)${NC}"
                else
                    echo -e "${RED}  ✗ 推薦數: 預期 >=$expected_rec, 實際 $actual_rec${NC}"
                    test_passed=false
                fi
                continue
            fi

            # Parse item[@index]:field=value
            local item_part="${check%%:*}"
            local field_value="${check#*:}"
            local field="${field_value%%=*}"
            local expected="${field_value#*=}"

            # Extract optional @index (e.g. 琥珀鐵觀音@1 → index 1)
            local item_name match_index=0
            if [[ "$item_part" == *@* ]]; then
                item_name="${item_part%%@*}"
                match_index="${item_part#*@}"
            else
                item_name="$item_part"
            fi

            # Find the order by item name (use match_index-th match)
            local order_json
            order_json=$(echo "$response" | jq -c "[.orders[] | select(.item == \"$item_name\")] | .[$match_index]")

            if [ "$order_json" == "null" ] || [ -z "$order_json" ]; then
                echo -e "${RED}  ✗ 找不到品項「$item_name」${NC}"
                test_passed=false
                continue
            fi

            local actual
            case "$field" in
                category) actual=$(echo "$order_json" | jq -r '.category // "null"') ;;
                ice)      actual=$(echo "$order_json" | jq -r '.ice // "null"') ;;
                sugar)    actual=$(echo "$order_json" | jq -r '.sugar // "null"') ;;
                size)     actual=$(echo "$order_json" | jq -r '.size // "null"') ;;
                qty)      actual=$(echo "$order_json" | jq -r '.qty // 1') ;;
                addon)    actual=$(echo "$order_json" | jq -r '.addon | join(",")') ;;
                not_on_menu) actual=$(echo "$order_json" | jq -r '.not_on_menu // false') ;;
            esac

            if [ "$actual" == "$expected" ]; then
                echo -e "${GREEN}  ✓ $item_name: $field=$actual${NC}"
            else
                echo -e "${RED}  ✗ $item_name: $field 預期 $expected, 實際 $actual${NC}"
                test_passed=false
            fi
        done
    fi

    if [ "$test_passed" = true ]; then
        ((PASS++))
    else
        ((FAIL++))
        echo "$response" | jq '.' 2>/dev/null
    fi
}

echo "========================================"
echo "  泛化測試 - 全新自然語言表達"
echo "========================================"

# ==================== 1. 自然口語點單 ====================
echo -e "\n${BLUE}【一、自然口語 — 非標準句式】${NC}"

test_order "口語：老闆開頭" \
    "老闆，三窨十五茉一杯" \
    "1" \
    "三窨十五茉:category=pure_tea|intent=order"

test_order "口語：請給我" \
    "請給我一杯琥珀鐵觀音，少冰半糖" \
    "1" \
    "琥珀鐵觀音:ice=少冰|琥珀鐵觀音:sugar=五分糖"

test_order "口語：想喝...的" \
    "想喝桂花的，熱的無糖" \
    "1" \
    "桂花輕烏龍:ice=熱飲|桂花輕烏龍:sugar=無糖"

test_order "口語：可以給我...嗎" \
    "可以給我一杯朱槿普洱紅嗎，微糖少冰" \
    "1" \
    "朱槿普洱紅:sugar=三分糖|朱槿普洱紅:ice=少冰"

# ==================== 2. 縮寫 + 客製化組合 ====================
echo -e "\n${BLUE}【二、縮寫 + 客製化 — 新組合】${NC}"

test_order "縮寫：普洱+大杯鮮奶" \
    "普洱鮮奶大杯去冰" \
    "1" \
    "朱槿普洱紅:category=fresh_milk_tea|朱槿普洱紅:size=L|朱槿普洱紅:ice=去冰"

test_order "縮寫：玉露+奶茶" \
    "玉露奶茶中杯微糖" \
    "1" \
    "米香玉露菁:category=rich_milk_tea|米香玉露菁:size=M|米香玉露菁:sugar=三分糖"

test_order "縮寫：蜜香紅+加料" \
    "蜜香紅加茶凍，半糖" \
    "1" \
    "金杯蜜香紅:addon=紅烏龍茶凍|金杯蜜香紅:sugar=五分糖"

test_order "縮寫：紅烏龍 → 茶花紅烏龍" \
    "紅烏龍奶茶半糖少冰大杯" \
    "1" \
    "茶花紅烏龍:category=rich_milk_tea|茶花紅烏龍:sugar=五分糖|茶花紅烏龍:ice=少冰|茶花紅烏龍:size=L"

# ==================== 3. 多品項 — 新搭配 ====================
echo -e "\n${BLUE}【三、多品項 — 未見過的搭配】${NC}"

test_order "兩品項：純茶+果茶" \
    "朱槿普洱紅純茶一杯，還有一杯青檸桂蜜" \
    "2" \
    "朱槿普洱紅:category=pure_tea|青檸桂蜜:category=fruit_tea"

test_order "兩品項：鮮奶+加料" \
    "金杯蜜香紅鮮奶中杯半糖，加一杯琥珀鐵觀音鮮奶大杯去冰加燕麥粒" \
    "2" \
    "金杯蜜香紅:category=fresh_milk_tea|琥珀鐵觀音:category=fresh_milk_tea|琥珀鐵觀音:addon=燕麥粒"

test_order "三品項：不同類別各一" \
    "米香玉露菁純茶、琥珀鐵觀音奶茶中杯、蜜桃紅烏龍" \
    "3" \
    "米香玉露菁:category=pure_tea|琥珀鐵觀音:category=rich_milk_tea|蜜桃紅烏龍:category=fruit_tea"

# ==================== 4. 果茶 ColdOnly 驗證 ====================
echo -e "\n${BLUE}【四、果茶限制 — 冷飲/單一尺寸】${NC}"

test_order "果茶：蜜桃紅烏龍（不應有size）" \
    "蜜桃紅烏龍無糖" \
    "1" \
    "蜜桃紅烏龍:category=fruit_tea|蜜桃紅烏龍:size=null"

test_order "果茶：葡柚十五茉（不應有size）" \
    "葡柚十五茉一杯少冰" \
    "1" \
    "葡柚十五茉:category=fruit_tea|葡柚十五茉:size=null"

test_order "果茶：要求熱飲應被修正" \
    "青檸桂蜜熱的" \
    "1" \
    "青檸桂蜜:category=fruit_tea|青檸桂蜜:ice=正常冰"

# ==================== 5. 跨分類 — 同品項不同分類 ====================
echo -e "\n${BLUE}【五、跨分類 — 同品項不同分類】${NC}"

test_order "跨分類：三窨十五茉 純茶+奶茶" \
    "三窨十五茉純茶無糖一杯，再一杯三窨十五茉奶茶半糖" \
    "2" \
    "intent=order"

test_order "跨分類：桂花 鮮奶+奶蓋" \
    "桂花輕烏龍鮮奶大杯去冰；再來桂花輕烏龍奶蓋一杯" \
    "2" \
    "intent=order"

test_order "跨分類：茶花紅烏龍 純茶+奶茶" \
    "一杯茶花紅烏龍純茶，另外要一杯茶花紅烏龍奶茶少冰" \
    "2" \
    "intent=order"

# ==================== 6. 推薦 & 閒聊 ====================
echo -e "\n${BLUE}【六、推薦 & 閒聊】${NC}"

test_order "推薦：新口語" \
    "你們什麼茶比較好喝" \
    "0" \
    "intent=recommend|rec_count=1"

test_order "推薦：場景型" \
    "天氣好熱想喝冰的，推薦一個" \
    "0" \
    "intent=recommend|rec_count=1"

test_order "閒聊：營業時間" \
    "你們幾點打烊" \
    "0" \
    "intent=other"

test_order "閒聊：其他問題" \
    "可以用信用卡嗎" \
    "0" \
    "intent=other"

# ==================== 7. 修改 & 猶豫 ====================
echo -e "\n${BLUE}【七、修改/猶豫/取消】${NC}"

test_order "修改品項" \
    "桂花輕烏龍...不對不對，換成琥珀鐵觀音好了" \
    "1" \
    "琥珀鐵觀音:qty=1"

test_order "修改甜度" \
    "三窨十五茉半糖...啊還是無糖好了" \
    "1" \
    "三窨十五茉:sugar=無糖"

test_order "猶豫後決定" \
    "嗯...琥珀鐵觀音跟桂花輕烏龍我想想...好吧兩個都要" \
    "2" \
    "琥珀鐵觀音:qty=1|桂花輕烏龍:qty=1"

# ==================== 8. 加料組合 ====================
echo -e "\n${BLUE}【八、加料 — 各種表達】${NC}"

test_order "加料：加茶凍和燕麥" \
    "米香玉露菁鮮奶加茶凍和燕麥粒" \
    "1" \
    "米香玉露菁:addon=紅烏龍茶凍,燕麥粒"

test_order "加料：不存在的加料" \
    "桂花輕烏龍加珍珠" \
    "1" \
    "桂花輕烏龍:qty=1"

test_order "加料：只加燕麥" \
    "琥珀鐵觀音奶茶大杯加燕麥粒半糖去冰" \
    "1" \
    "琥珀鐵觀音:addon=燕麥粒|琥珀鐵觀音:category=rich_milk_tea|琥珀鐵觀音:size=L"

# ==================== 9. 數量表達 ====================
echo -e "\n${BLUE}【九、數量 — 各種表達方式】${NC}"

test_order "數量：三杯同品項" \
    "三窨十五茉三杯" \
    "1" \
    "三窨十五茉:qty=3"

test_order "數量：各一杯" \
    "金杯蜜香紅跟朱槿普洱紅各一杯" \
    "2" \
    "金杯蜜香紅:qty=1|朱槿普洱紅:qty=1"

test_order "數量：多杯+客製" \
    "琥珀鐵觀音鮮奶兩杯，都要去冰半糖大杯" \
    "1" \
    "琥珀鐵觀音:qty=2|琥珀鐵觀音:ice=去冰|琥珀鐵觀音:sugar=五分糖|琥珀鐵觀音:size=L"

# ==================== 10. 混合場景 — 點單+閒聊 ====================
echo -e "\n${BLUE}【十、混合場景 — 點單夾雜閒聊】${NC}"

test_order "混合：先閒聊後點單" \
    "你們茶葉是哪裡來的啊，給我一杯三窨十五茉好了" \
    "1" \
    "三窨十五茉:qty=1|intent=order"

test_order "混合：點單帶感嘆" \
    "好渴喔，趕快給我一杯金杯蜜香紅鮮奶大杯去冰" \
    "1" \
    "金杯蜜香紅:category=fresh_milk_tea|金杯蜜香紅:size=L|金杯蜜香紅:ice=去冰"

# ==================== 11. 非菜單品項 ====================
echo -e "\n${BLUE}【十一、非菜單品項】${NC}"

test_order "非菜單問句 → 推薦替代品" \
    "有抹茶拿鐵嗎" \
    "0" \
    "intent=recommend|rec_count=1"

test_order "非菜單：烏龍綠" \
    "一杯烏龍綠" \
    "1" \
    "intent=order"

# ==================== 12. 多輪對話（分號分隔）====================
echo -e "\n${BLUE}【十二、多輪對話 — 分號分隔】${NC}"

test_order "多輪：先點再加" \
    "一杯朱槿普洱紅鮮奶中杯；再加一杯米香玉露菁純茶無糖" \
    "2" \
    "朱槿普洱紅:category=fresh_milk_tea|米香玉露菁:category=pure_tea|米香玉露菁:sugar=無糖"

test_order "多輪：果茶+奶茶" \
    "蜜桃紅烏龍一杯；再來一杯琥珀鐵觀音奶茶半糖少冰大杯" \
    "2" \
    "蜜桃紅烏龍:category=fruit_tea|琥珀鐵觀音:category=rich_milk_tea|琥珀鐵觀音:size=L"

test_order "多輪：三段" \
    "桂花輕烏龍純茶；再一杯桂花輕烏龍鮮奶去冰；還要一杯茶花紅烏龍奶蓋" \
    "3" \
    "intent=order"

# ==================== 13. 易混淆品項 ====================
echo -e "\n${BLUE}【十三、易混淆品項共存】${NC}"

test_order "混淆：金杯蜜香紅 vs 琥珀鐵觀音（同為鮮奶/奶茶雙分類）" \
    "金杯蜜香紅鮮奶一杯，琥珀鐵觀音奶茶一杯" \
    "2" \
    "金杯蜜香紅:category=fresh_milk_tea|琥珀鐵觀音:category=rich_milk_tea"

test_order "混淆：蜜桃紅烏龍 vs 紅烏龍奶蓋" \
    "一杯蜜桃紅烏龍，再一杯茶花紅烏龍奶蓋" \
    "2" \
    "蜜桃紅烏龍:category=fruit_tea|茶花紅烏龍:category=cloud_velvet"

test_order "混淆：葡柚十五茉 vs 三窨十五茉" \
    "葡柚十五茉和三窨十五茉純茶各一杯" \
    "2" \
    "葡柚十五茉:category=fruit_tea|三窨十五茉:category=pure_tea"

# ==================== 14. 熱飲 ====================
echo -e "\n${BLUE}【十四、熱飲各種表達】${NC}"

test_order "熱飲：直說熱的" \
    "金杯蜜香紅熱的" \
    "1" \
    "金杯蜜香紅:ice=熱飲"

test_order "熱飲：要熱飲" \
    "三窨十五茉純茶要熱飲無糖" \
    "1" \
    "三窨十五茉:ice=熱飲|三窨十五茉:sugar=無糖|三窨十五茉:category=pure_tea"

test_order "熱飲：溫的（應為熱飲或其他處理）" \
    "桂花輕烏龍溫的" \
    "1" \
    "桂花輕烏龍:qty=1"

# ==================== 15. 壓力測試 ====================
echo -e "\n${BLUE}【十五、壓力 — 長句/多品項】${NC}"

test_order "壓力：4品項一次" \
    "三窨十五茉純茶、桂花輕烏龍鮮奶大杯去冰、琥珀鐵觀音奶茶半糖、蜜桃紅烏龍，通通一杯" \
    "4" \
    "三窨十五茉:category=pure_tea|桂花輕烏龍:category=fresh_milk_tea|琥珀鐵觀音:category=rich_milk_tea|蜜桃紅烏龍:category=fruit_tea"

test_order "壓力：完整客製化" \
    "朱槿普洱紅鮮奶大杯三分糖微冰加紅烏龍茶凍" \
    "1" \
    "朱槿普洱紅:category=fresh_milk_tea|朱槿普洱紅:size=L|朱槿普洱紅:sugar=三分糖|朱槿普洱紅:ice=微冰|朱槿普洱紅:addon=紅烏龍茶凍"

# ==================== 16. 雙份加料 ====================
echo -e "\n${BLUE}【十六、雙份加料】${NC}"

test_order "雙份加料：雙份茶凍" \
    "琥珀鐵觀音鮮奶茶大杯半糖少冰加雙份茶凍" \
    "1" \
    "琥珀鐵觀音:category=fresh_milk_tea|琥珀鐵觀音:addon=紅烏龍茶凍,紅烏龍茶凍|琥珀鐵觀音:size=L"

test_order "雙份加料：兩份燕麥" \
    "金杯蜜香紅奶茶加兩份燕麥粒半糖少冰" \
    "1" \
    "金杯蜜香紅:category=rich_milk_tea|金杯蜜香紅:addon=燕麥粒,燕麥粒"

test_order "雙份加料：雙份+單份混合" \
    "桂花輕烏龍鮮奶茶中杯無糖去冰加雙份茶凍和燕麥" \
    "1" \
    "桂花輕烏龍:category=fresh_milk_tea|桂花輕烏龍:addon=紅烏龍茶凍,紅烏龍茶凍,燕麥粒"

# ==================== 17. 多輪 category 獨立性 ====================
echo -e "\n${BLUE}【十七、多輪 category 不沿用】${NC}"

test_order "多輪：1st 鮮奶 → 2nd 不同品項未指定（不該沿用 fresh_milk_tea）" \
    "琥珀鐵觀音鮮奶茶半糖少冰；再一杯金杯蜜香紅無糖去冰" \
    "2" \
    "琥珀鐵觀音:category=fresh_milk_tea|金杯蜜香紅:category=rich_milk_tea"

test_order "多輪：1st 奶蓋 → 2nd 不同品項未指定（不該沿用 cloud_velvet）" \
    "茶花紅烏龍奶蓋；再一杯桂花輕烏龍半糖少冰" \
    "2" \
    "茶花紅烏龍:category=cloud_velvet|桂花輕烏龍:category=pure_tea"

test_order "多輪：1st 未指定 → 2nd 指定鮮奶（應正確辨識）" \
    "金杯蜜香紅一杯半糖少冰；再一杯琥珀鐵觀音鮮奶大杯去冰" \
    "2" \
    "金杯蜜香紅:category=rich_milk_tea|琥珀鐵觀音:category=fresh_milk_tea|琥珀鐵觀音:size=L"

test_order "多輪：1st 純茶 → 2nd 不同品項未指定（不該沿用 pure_tea）" \
    "三窨十五茉純茶無糖去冰；再一杯米香玉露菁半糖少冰" \
    "2" \
    "三窨十五茉:category=pure_tea|米香玉露菁:category=pure_tea"

# ==================== 18. 數量拆分 ====================
echo -e "\n${BLUE}【十八、數量拆分（分類/客製）】${NC}"

test_order "分類拆分：三杯桂花輕烏龍，兩杯奶茶一杯純茶（qty合併→2筆）" \
    "三杯桂花輕烏龍，兩杯奶茶半糖少冰，一杯純茶無糖去冰" \
    "2" \
    "桂花輕烏龍:category=rich_milk_tea"

test_order "分類拆分：兩杯琥珀鐵觀音，鮮奶茶+奶茶" \
    "兩杯琥珀鐵觀音，鮮奶茶的大杯半糖去冰，奶茶的無糖少冰" \
    "2" \
    "琥珀鐵觀音:category=fresh_milk_tea|琥珀鐵觀音@1:category=rich_milk_tea"

test_order "客製拆分：三杯三窨十五茉鮮奶茶，兩杯半糖一杯無糖（qty合併→2筆）" \
    "三杯三窨十五茉鮮奶茶少冰，兩杯半糖一杯無糖" \
    "2" \
    "三窨十五茉:category=fresh_milk_tea"

echo ""
echo "========================================"
echo "  測試結果總結"
echo "========================================"
echo -e "${GREEN}通過: $PASS${NC}"
echo -e "${RED}失敗: $FAIL${NC}"
echo "總計: $TOTAL"
if [ $TOTAL -gt 0 ]; then
    echo "通過率: $(( PASS * 100 / TOTAL ))%"
fi
echo ""
if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ 所有測試通過！${NC}"
else
    echo -e "${RED}✗ 有 $FAIL 個測試失敗${NC}"
fi
