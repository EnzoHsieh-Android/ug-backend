package service

import "testing"

func TestNormalizeSTT(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// === Layer 1: Exact table (known typos) ===
		{"exact: 三因→三窨", "三因十五茉鮮奶半糖少冰", "三窨十五茉鮮奶半糖少冰"},
		{"exact: 湖泊→琥珀", "湖泊鐵觀音鮮奶半糖少冰", "琥珀鐵觀音鮮奶半糖少冰"},
		{"exact: 伴糖→半糖", "三窨十五茉鮮奶伴糖少冰", "三窨十五茉鮮奶半糖少冰"},
		{"exact: 趣冰→去冰", "桂花輕烏龍鮮奶半糖趣冰", "桂花輕烏龍鮮奶半糖去冰"},
		{"exact: 涼杯→兩杯", "涼杯三窨十五茉鮮奶半糖少冰", "兩杯三窨十五茉鮮奶半糖少冰"},

		// === Layer 2: Pinyin fuzzy (UNKNOWN typos not in exact table) ===
		// These are new typos the exact table has never seen.
		{"fuzzy: 糊珀鐵觀音", "糊珀鐵觀音鮮奶半糖少冰", "琥珀鐵觀音鮮奶半糖少冰"},
		{"fuzzy: 桂花傾烏龍", "桂花傾烏龍鮮奶半糖少冰", "桂花輕烏龍鮮奶半糖少冰"},
		{"fuzzy: 迷桃紅烏龍", "迷桃紅烏龍半糖少冰", "蜜桃紅烏龍半糖少冰"},
		{"fuzzy: 朱僅普洱紅", "朱僅普洱紅去冰無糖", "朱槿普洱紅去冰無糖"},
		{"fuzzy: 半糖 correct stays", "三窨十五茉鮮奶半糖少冰", "三窨十五茉鮮奶半糖少冰"},

		// === Combined: exact + fuzzy ===
		{"combo: 三因+unknown甜度", "三因十五茉鮮奶伴糖少冰", "三窨十五茉鮮奶半糖少冰"},

		// === Should NOT replace correct text ===
		{"no-op: correct input", "琥珀鐵觀音鮮奶半糖少冰", "琥珀鐵觀音鮮奶半糖少冰"},
		{"no-op: correct menu item", "桂花輕烏龍純茶去冰無糖", "桂花輕烏龍純茶去冰無糖"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSTT(tt.input)
			if got != tt.want {
				t.Errorf("\n  input: %s\n  got:   %s\n  want:  %s", tt.input, got, tt.want)
			}
		})
	}
}
