package service

import (
	"strings"

	"github.com/xiechengru/ug-backend/model"
)

// Menu data - single source of truth
var menu = &model.Menu{
	Categories: []model.MenuCategory{
		{
			ID:   "pure_tea",
			Name: "研選純茶",
			Items: []model.MenuItem{
				{Name: "三窨十五茉", Category: "pure_tea", PriceL: 45, HasSize: false,
					Description: "經三次窨花的茉莉綠茶，花香清雅帶甘甜，回韻悠長",
					Tags:        []string{"花香", "清爽", "不甜", "經典", "飯後"}},
				{Name: "桂花輕烏龍", Category: "pure_tea", PriceL: 45, HasSize: false,
					Description: "桂花與輕發酵烏龍結合，花果蜜韻輕柔，入口順滑",
					Tags:        []string{"花香", "清爽", "淡雅", "女性", "下午茶"}},
				{Name: "茶花紅烏龍", Category: "pure_tea", PriceL: 45, HasSize: false,
					Description: "重焙火紅烏龍，帶天然花香與熟果甜韻，茶味厚實",
					Tags:        []string{"茶味重", "醇厚", "焙火", "熱飲推薦", "秋冬"}},
				{Name: "朱槿普洱紅", Category: "pure_tea", PriceL: 40, HasSize: false,
					Description: "洛神花搭配普洱紅茶，微酸回甘，解膩開胃",
					Tags:        []string{"微酸", "解膩", "飯後", "開胃", "夏天"}},
				{Name: "米香玉露菁", Category: "pure_tea", PriceL: 45, HasSize: false,
					Description: "日式玉露菁茶帶米穀香氣，鮮爽細膩，餘韻回甘",
					Tags:        []string{"清爽", "米香", "日式", "鮮爽", "第一次嘗試"}},
			},
		},
		{
			ID:   "fresh_milk_tea",
			Name: "牧場直達",
			Items: []model.MenuItem{
				{Name: "三窨十五茉", Category: "fresh_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "茉莉花香融合鮮乳，清香滑順，甜度適中最對味",
					Tags:        []string{"花香", "鮮奶", "清爽", "人氣", "百搭"}},
				{Name: "桂花輕烏龍", Category: "fresh_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "桂花烏龍配鮮乳，蜜韻與奶香交融，溫潤順口",
					Tags:        []string{"花香", "鮮奶", "溫潤", "女性", "下午茶"}},
				{Name: "茶花烏龍", Category: "fresh_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "烏龍茶韻搭配鮮乳，茶奶平衡，口感圓潤",
					Tags:        []string{"鮮奶", "圓潤", "經典", "百搭"}},
				{Name: "琥珀鐵觀音", Category: "fresh_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "重焙鐵觀音遇上鮮乳，焙火香濃郁，奶茶控首選",
					Tags:        []string{"鮮奶", "茶味重", "焙火", "濃郁", "配甜點"}},
				{Name: "金杯蜜香紅", Category: "fresh_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "蜜香紅茶與鮮乳，天然蜜甜與奶香完美搭配",
					Tags:        []string{"鮮奶", "蜜甜", "紅茶", "平價", "第一次嘗試"}},
				{Name: "朱槿普洱紅", Category: "fresh_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "洛神普洱配鮮乳，微酸奶香交織，風味獨特",
					Tags:        []string{"鮮奶", "微酸", "獨特", "解膩"}},
				{Name: "米香玉露菁", Category: "fresh_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "玉露菁米香融入鮮乳，穀物奶香，溫暖療癒",
					Tags:        []string{"鮮奶", "米香", "溫暖", "療癒"}},
			},
		},
		{
			ID:   "rich_milk_tea",
			Name: "UG奶茶",
			Items: []model.MenuItem{
				{Name: "三窨十五茉", Category: "rich_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "茉莉花茶搭配奶茶底，花香茶韻與奶味層次豐富",
					Tags:        []string{"花香", "奶茶", "層次", "人氣"}},
				{Name: "桂花輕烏龍", Category: "rich_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "桂花烏龍奶茶，蜜韻奶香濃厚，甜而不膩",
					Tags:        []string{"花香", "奶茶", "濃厚", "甜而不膩"}},
				{Name: "茶花紅烏龍", Category: "rich_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "紅烏龍奶茶，焙火茶香搭奶底，醇厚飽滿",
					Tags:        []string{"奶茶", "茶味重", "醇厚", "焙火"}},
				{Name: "琥珀鐵觀音", Category: "rich_milk_tea", PriceM: 65, PriceL: 75, HasSize: true,
					Description: "鐵觀音奶茶，重焙火香氣搭配奶底更顯濃郁",
					Tags:        []string{"奶茶", "茶味重", "焙火", "濃郁", "經典"}},
				{Name: "金杯蜜香紅", Category: "rich_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "蜜香紅茶奶茶，自然蜜甜奶滑，老少咸宜",
					Tags:        []string{"奶茶", "蜜甜", "紅茶", "平價"}},
				{Name: "朱槿普洱紅", Category: "rich_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "洛神普洱奶茶，微酸奶香特殊風味，嘗鮮首選",
					Tags:        []string{"奶茶", "微酸", "獨特", "嘗鮮"}},
				{Name: "米香玉露菁", Category: "rich_milk_tea", PriceM: 60, PriceL: 70, HasSize: true,
					Description: "玉露菁奶茶，米穀香與奶茶融合，溫潤舒心",
					Tags:        []string{"奶茶", "米香", "溫潤", "舒心"}},
			},
		},
		{
			ID:   "fruit_tea",
			Name: "果香四溢",
			Items: []model.MenuItem{
				{Name: "青檸桂蜜", Category: "fruit_tea", PriceM: 65, HasSize: false, ColdOnly: true,
					Description: "青檸搭配桂花蜜，酸甜清爽，夏日消暑聖品",
					Tags:        []string{"果茶", "酸甜", "清爽", "夏天", "消暑"}},
				{Name: "蜜桃紅烏龍", Category: "fruit_tea", PriceM: 75, HasSize: false, ColdOnly: true,
					Description: "水蜜桃果香與紅烏龍茶韻交織，甜蜜芬芳",
					Tags:        []string{"果茶", "甜", "水果", "女性", "人氣"}},
				{Name: "葡柚十五茉", Category: "fruit_tea", PriceM: 80, HasSize: false, ColdOnly: true,
					Description: "葡萄柚酸香配茉莉花茶，清新微苦，大人味果茶",
					Tags:        []string{"果茶", "微苦", "清新", "大人味", "獨特"}},
			},
		},
		{
			ID:   "cloud_velvet",
			Name: "雲頂奶蓋",
			Items: []model.MenuItem{
				{Name: "桂花輕烏龍", Category: "cloud_velvet", PriceM: 95, HasSize: false,
					Description: "桂花烏龍上覆綿密奶蓋，鹹甜交融，口感豐富",
					Tags:        []string{"奶蓋", "花香", "鹹甜", "豐富", "招牌"}},
				{Name: "茶花紅烏龍", Category: "cloud_velvet", PriceM: 95, HasSize: false,
					Description: "紅烏龍焙火韻搭奶蓋，茶香奶鹹完美平衡",
					Tags:        []string{"奶蓋", "茶味重", "焙火", "招牌", "配甜點"}},
			},
		},
	},
	Options: model.MenuOptions{
		Sugar: []string{"全糖", "五分糖", "三分糖", "一分糖", "無糖", "正常糖", "半糖", "少糖", "微糖"},
		Ice:   []string{"正常冰", "少冰", "微冰", "去冰", "熱飲"},
		Size:  []string{"M", "L"},
		Addon: []model.Addon{
			{Name: "紅烏龍茶凍", Price: 15},
			{Name: "燕麥粒", Price: 15},
		},
	},
}

// GetMenu returns the complete menu
func GetMenu() *model.Menu {
	return menu
}

// GetMenuItem returns a single menu item by name
func GetMenuItem(name string) *model.MenuItem {
	for _, cat := range menu.Categories {
		for _, item := range cat.Items {
			if item.Name == name {
				return &item
			}
		}
	}
	return nil
}

// IsValidMenuItem checks if an item name is on the menu
func IsValidMenuItem(name string) bool {
	return GetMenuItem(name) != nil
}

// ============================================================
// Fuzzy Item Matching
// ============================================================

// Category suffixes to strip when matching
var categorySuffixes = []string{"鮮奶", "奶茶", "純茶", "奶蓋", "果茶"}

// Option suffixes to strip (sweetness, ice, temperature)
var optionSuffixes = []string{
	"熱", "熱的", "溫",
	"無糖", "一分糖", "三分糖", "五分糖", "全糖", "微糖", "半糖", "少糖", "正常糖",
	"去冰", "微冰", "少冰", "正常冰",
}

// Abbreviation to full name mapping
var abbreviationMap = map[string]string{
	// 三窨十五茉
	"茉莉":   "三窨十五茉",
	"十五茉":  "三窨十五茉",
	"茉莉熱":  "三窨十五茉",
	"茉莉無糖": "三窨十五茉",
	// 桂花輕烏龍
	"桂花":    "桂花輕烏龍",
	"輕烏龍":   "桂花輕烏龍",
	"桂花烏龍":  "桂花輕烏龍",
	"桂花熱":   "桂花輕烏龍",
	"桂花無糖":  "桂花輕烏龍",
	// 琥珀鐵觀音
	"鐵觀音":   "琥珀鐵觀音",
	"鐵觀音熱":  "琥珀鐵觀音",
	"鐵觀音無糖": "琥珀鐵觀音",
	// 金杯蜜香紅
	"蜜香紅":   "金杯蜜香紅",
	"金杯":    "金杯蜜香紅",
	"金杯熱":   "金杯蜜香紅",
	"金杯無糖":  "金杯蜜香紅",
	// 朱槿普洱紅
	"普洱":    "朱槿普洱紅",
	"普洱紅":   "朱槿普洱紅",
	"普洱熱":   "朱槿普洱紅",
	"普洱無糖":  "朱槿普洱紅",
	// 米香玉露菁
	"紅玉":    "紅玉紅茶",
	"玉露":    "米香玉露菁",
	"玉露菁":   "米香玉露菁",
	"玉露熱":   "米香玉露菁",
	"玉露無糖":  "米香玉露菁",
	// 果茶
	"蜜桃":    "蜜桃紅烏龍",
	"蜜桃烏龍":  "蜜桃紅烏龍",
	"葡柚":    "葡柚十五茉",
	"青檸":    "青檸桂蜜",
	"桂蜜":    "青檸桂蜜",
	// 其他
	"茶花":    "茶花烏龍",
	"紅烏龍":   "茶花紅烏龍",
}

// GetAllMenuItemNames returns all unique item names from the menu
func GetAllMenuItemNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, cat := range menu.Categories {
		for _, item := range cat.Items {
			if !seen[item.Name] {
				seen[item.Name] = true
				names = append(names, item.Name)
			}
		}
	}
	return names
}

// FuzzyMatchMenuItem tries to find the best matching menu item for a given name.
// Returns (matchedName, matched). If no match found, returns (original, false).
func FuzzyMatchMenuItem(name string) (string, bool) {
	// 1. Direct match - already valid
	if IsValidMenuItem(name) {
		return name, true
	}

	// 2. Strip category suffix and try again
	stripped := name
	for _, suffix := range categorySuffixes {
		if strings.HasSuffix(name, suffix) {
			stripped = strings.TrimSuffix(name, suffix)
			break
		}
	}

	// Check if stripped name is valid
	if stripped != name && IsValidMenuItem(stripped) {
		return stripped, true
	}

	// 2.5. Strip option suffixes (熱、無糖、去冰 etc.)
	for _, suffix := range optionSuffixes {
		if strings.HasSuffix(stripped, suffix) {
			stripped = strings.TrimSuffix(stripped, suffix)
			break
		}
	}

	// Check if stripped name is valid after removing option suffix
	if IsValidMenuItem(stripped) {
		return stripped, true
	}

	// 3. Check abbreviation map (with and without suffix)
	if fullName, ok := abbreviationMap[stripped]; ok {
		return fullName, true
	}
	if fullName, ok := abbreviationMap[name]; ok {
		return fullName, true
	}

	// 4. Check if stripped name contains a menu item name
	menuItems := GetAllMenuItemNames()
	for _, menuItem := range menuItems {
		if strings.Contains(stripped, menuItem) {
			return menuItem, true
		}
		if strings.Contains(menuItem, stripped) && len(stripped) >= 2 {
			return menuItem, true
		}
	}

	// 5. Pinyin fuzzy matching as last resort
	if matched := fuzzyMatchByPinyin(stripped); matched != "" {
		return matched, true
	}

	return name, false
}

// fuzzyMatchByPinyin uses pinyin similarity to find the best matching menu item
// Only matches if the input is very similar to avoid false positives
func fuzzyMatchByPinyin(name string) string {
	if len(name) < 2 {
		return ""
	}

	namePy := toPinyin(name)
	if len(namePy) == 0 {
		return ""
	}

	menuItems := GetAllMenuItemNames()
	var bestMatch string
	bestSim := 0.0
	const threshold = 0.8 // High threshold to avoid false positives

	for _, menuItem := range menuItems {
		itemPy := toPinyin(menuItem)
		if len(itemPy) == 0 {
			continue
		}

		// Only consider if lengths are similar (within 2 syllables)
		lenDiff := len(namePy) - len(itemPy)
		if lenDiff < -2 || lenDiff > 2 {
			continue
		}

		// Calculate similarity based on matching syllables
		sim := pinyinContainsSimilarity(namePy, itemPy)
		if sim > bestSim && sim >= threshold {
			bestSim = sim
			bestMatch = menuItem
		}
	}

	return bestMatch
}

// pinyinContainsSimilarity checks if namePy syllables are contained in itemPy
// Returns similarity score (0.0 ~ 1.0)
func pinyinContainsSimilarity(namePy, itemPy []string) float64 {
	if len(namePy) == 0 {
		return 0
	}

	// Count how many syllables from namePy are found in itemPy
	matched := 0
	itemPySet := make(map[string]int)
	for _, py := range itemPy {
		itemPySet[py]++
	}

	for _, py := range namePy {
		if itemPySet[py] > 0 {
			matched++
			itemPySet[py]--
		}
	}

	return float64(matched) / float64(len(namePy))
}
