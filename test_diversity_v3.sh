#!/bin/bash
# 多樣性測試 v3 - 變體測試，驗證模型泛化能力
# 基於 v2 測試案例做口語變化

API_URL="http://localhost:3000/api/order"
PASS=0
FAIL=0
TOTAL=0

test_order() {
    local test_name="$1"
    local input="$2"
    local expected_count="$3"
    shift 3
    local expected_items=("$@")

    TOTAL=$((TOTAL + 1))

    echo ""
    echo -e "\033[1;33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
    echo -e "\033[1;33m測試: $test_name\033[0m"
    echo "輸入: $input"
    echo -e "\033[1;33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"

    response=$(curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"$input\"}")

    # Count orders
    actual_count=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('orders',[])))" 2>/dev/null)

    # Get items
    actual_items=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(o.get('item','')) for o in d.get('orders',[])]" 2>/dev/null)

    # Check not_on_menu
    has_not_on_menu=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(any(o.get('not_on_menu',False) for o in d.get('orders',[])))" 2>/dev/null)

    local test_passed=true

    # Check count
    if [ "$actual_count" = "$expected_count" ]; then
        echo -e "\033[0;32m✓ 訂單數量正確: $actual_count\033[0m"
    else
        echo -e "\033[0;31m✗ 訂單數量錯誤: 預期 $expected_count, 實際 $actual_count\033[0m"
        test_passed=false
    fi

    # Check items
    for item in "${expected_items[@]}"; do
        if [[ "$item" == "NOT_ON_MENU" ]]; then
            if [ "$has_not_on_menu" = "True" ]; then
                echo -e "\033[1;33m⚠ 有品項不在菜單上 (預期)\033[0m"
            fi
            continue
        fi
        if echo "$actual_items" | grep -q "$item"; then
            echo -e "\033[0;32m✓ 包含: $item\033[0m"
        else
            echo -e "\033[0;31m✗ 缺少: $item\033[0m"
            test_passed=false
        fi
    done

    # Check unexpected not_on_menu
    local expect_nom=false
    for item in "${expected_items[@]}"; do
        if [[ "$item" == "NOT_ON_MENU" ]]; then
            expect_nom=true
        fi
    done
    if [ "$expect_nom" = false ] && [ "$has_not_on_menu" = "True" ]; then
        echo -e "\033[0;31m✗ 有品項被標記為不在菜單上\033[0m"
        test_passed=false
    fi

    if [ "$test_passed" = true ]; then
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
    fi
}

echo "========================================"
echo "  多樣性測試 v3 - 變體泛化測試"
echo "========================================"

# ==========================================
echo -e "\n\033[0;34m【一：基本點單變體】\033[0m"
# ==========================================

test_order "來一杯" \
    "來一杯三窨十五茉" 1 "三窨十五茉"

test_order "給我一杯" \
    "給我一杯桂花輕烏龍" 1 "桂花輕烏龍"

test_order "我想喝" \
    "我想喝琥珀鐵觀音" 1 "琥珀鐵觀音"

test_order "幫我來" \
    "幫我來一杯金杯蜜香紅" 1 "金杯蜜香紅"

test_order "要一個" \
    "要一個青檸桂蜜" 1 "青檸桂蜜"

# ==========================================
echo -e "\n\033[0;34m【二：縮寫變體】\033[0m"
# ==========================================

test_order "茉莉+選項" \
    "茉莉無糖去冰" 1 "三窨十五茉"

test_order "桂花+選項" \
    "桂花微糖微冰" 1 "桂花輕烏龍"

test_order "鐵觀音+選項" \
    "鐵觀音半糖少冰" 1 "琥珀鐵觀音"

test_order "普洱+選項" \
    "普洱無糖去冰" 1 "朱槿普洱紅"

test_order "金杯+選項" \
    "金杯半糖少冰" 1 "金杯蜜香紅"

test_order "蜜桃+選項" \
    "蜜桃少糖" 1 "蜜桃紅烏龍"

# ==========================================
echo -e "\n\033[0;34m【三：加料變體】\033[0m"
# ==========================================

test_order "鐵觀音+茶凍半糖去冰" \
    "琥珀鐵觀音加紅烏龍茶凍半糖去冰" 1 "琥珀鐵觀音"

test_order "縮寫鐵觀音+茶凍" \
    "鐵觀音加茶凍微糖" 1 "琥珀鐵觀音"

test_order "茉莉+燕麥粒" \
    "三窨十五茉加燕麥粒" 1 "三窨十五茉"

test_order "桂花奶茶+茶凍" \
    "桂花輕烏龍奶茶加紅烏龍茶凍" 1 "桂花輕烏龍"

test_order "金杯+雙加料" \
    "金杯蜜香紅加紅烏龍茶凍和燕麥粒" 1 "金杯蜜香紅"

test_order "茶花紅烏龍+茶凍(易混淆)" \
    "茶花紅烏龍加紅烏龍茶凍半糖" 1 "茶花紅烏龍"

# ==========================================
echo -e "\n\033[0;34m【四：多品項變體】\033[0m"
# ==========================================

test_order "A跟B" \
    "三窨十五茉跟金杯蜜香紅" 2 "三窨十五茉" "金杯蜜香紅"

test_order "A還要B" \
    "琥珀鐵觀音，還要一杯桂花輕烏龍" 2 "琥珀鐵觀音" "桂花輕烏龍"

test_order "A再加B" \
    "朱槿普洱紅，再加一杯青檸桂蜜" 2 "朱槿普洱紅" "青檸桂蜜"

test_order "兩杯不同" \
    "一杯茉莉一杯桂花" 2 "三窨十五茉" "桂花輕烏龍"

test_order "三品項逗號分隔" \
    "茉莉，桂花，鐵觀音" 3 "三窨十五茉" "桂花輕烏龍" "琥珀鐵觀音"

test_order "四品項頓號" \
    "三窨十五茉、桂花輕烏龍、金杯蜜香紅、朱槿普洱紅" 4 "三窨十五茉" "桂花輕烏龍" "金杯蜜香紅" "朱槿普洱紅"

# ==========================================
echo -e "\n\033[0;34m【五：數量變體】\033[0m"
# ==========================================

test_order "兩杯同品" \
    "兩杯三窨十五茉" 1 "三窨十五茉"

test_order "五杯" \
    "五杯琥珀鐵觀音半糖" 1 "琥珀鐵觀音"

test_order "不對稱數量" \
    "兩杯茉莉和三杯桂花" 2 "三窨十五茉" "桂花輕烏龍"

# ==========================================
echo -e "\n\033[0;34m【六：修改/刪除變體】\033[0m"
# ==========================================

test_order "改品項 - 啊換成" \
    "金杯蜜香紅，啊，換成琥珀鐵觀音" 1 "琥珀鐵觀音"

test_order "改選項 - 改少冰" \
    "三窨十五茉去冰，啊改少冰" 1 "三窨十五茉"

test_order "取消其中一項" \
    "茉莉和桂花，桂花取消" 1 "三窨十五茉"

test_order "加料後取消加料" \
    "金杯蜜香紅加燕麥粒，燕麥粒不要了" 1 "金杯蜜香紅"

# ==========================================
echo -e "\n\033[0;34m【七：口語/閒聊變體】\033[0m"
# ==========================================

test_order "語氣詞開頭" \
    "嗯，那就茉莉好了" 1 "三窨十五茉"

test_order "請問+點單" \
    "請問有桂花嗎？那我要一杯" 1 "桂花輕烏龍"

test_order "幫朋友點" \
    "我跟我朋友各一杯，我茉莉他桂花" 2 "三窨十五茉" "桂花輕烏龍"

test_order "思考後決定" \
    "讓我想想...好，金杯蜜香紅無糖" 1 "金杯蜜香紅"

# ==========================================
echo -e "\n\033[0;34m【八：尺寸/類別變體】\033[0m"
# ==========================================

test_order "中杯鮮奶" \
    "中杯琥珀鐵觀音鮮奶" 1 "琥珀鐵觀音"

test_order "大杯奶茶" \
    "大杯金杯蜜香紅奶茶" 1 "金杯蜜香紅"

test_order "鮮奶+選項" \
    "桂花輕烏龍鮮奶半糖少冰" 1 "桂花輕烏龍"

test_order "奶蓋+選項" \
    "琥珀鐵觀音奶蓋微糖" 1 "琥珀鐵觀音"

# ==========================================
echo -e "\n\033[0;34m【九：非菜單變體】\033[0m"
# ==========================================

test_order "非菜單 - 綠茶" \
    "一杯綠茶" 1 "NOT_ON_MENU"

test_order "非菜單 - 咖啡" \
    "我要一杯拿鐵咖啡" 1 "NOT_ON_MENU"

# ==========================================

echo ""
echo "========================================"
echo "  測試結果總結"
echo "========================================"
echo -e "\033[0;32m通過: $PASS\033[0m"
echo -e "\033[0;31m失敗: $FAIL\033[0m"
echo "總計: $TOTAL"
echo "通過率: $((PASS * 100 / TOTAL))%"

if [ $FAIL -gt 0 ]; then
    echo -e "\n\033[1;33m⚠ 有 $FAIL 個測試失敗\033[0m"
fi
