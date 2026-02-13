package service

import (
	"strings"
	"unicode/utf8"

	"github.com/mozillazg/go-pinyin"
)

// ============================================================
// Layer 1: Exact replacement table (fast path, known typos)
// ============================================================

var exactReplacements = []struct{ old, new string }{
	// 品名 - 三窨十五茉 (常見誤聽)
	{"三因十五茉", "三窨十五茉"},
	{"三蔭十五茉", "三窨十五茉"},
	{"三印十五茉", "三窨十五茉"},
	{"三音十五茉", "三窨十五茉"},
	{"三殷十五茉", "三窨十五茉"},
	{"三英十五茉", "三窨十五茉"},
	{"三英四五茉", "三窨十五茉"},
	{"三因四五茉", "三窨十五茉"},
	{"三音四五茉", "三窨十五茉"},
	{"三蔭四五茉", "三窨十五茉"},
	{"三陰十五茉", "三窨十五茉"},
	{"三陰四五茉", "三窨十五茉"},
	{"三贏十五茉", "三窨十五茉"},
	{"三贏四五茉", "三窨十五茉"},
	{"湖泊鐵觀音", "琥珀鐵觀音"},
	{"虎魄鐵觀音", "琥珀鐵觀音"},
	{"湖泊鐵觀因", "琥珀鐵觀音"},
	{"虎魄鐵觀因", "琥珀鐵觀音"},
	{"鐵觀因", "鐵觀音"},
	{"珠槿普洱紅", "朱槿普洱紅"},
	{"朱瑾普洱紅", "朱槿普洱紅"},
	{"朱謹普洱紅", "朱槿普洱紅"},
	{"珠瑾普洱紅", "朱槿普洱紅"},
	{"貴花輕烏龍", "桂花輕烏龍"},
	{"貴花輕烏隆", "桂花輕烏龍"},
	{"桂花輕烏隆", "桂花輕烏龍"},
	{"迷香玉露菁", "米香玉露菁"},
	{"迷香玉露青", "米香玉露菁"},
	{"米香玉露青", "米香玉露菁"},
	{"密桃紅烏龍", "蜜桃紅烏龍"},
	{"密桃紅烏隆", "蜜桃紅烏龍"},
	{"蜜桃紅烏隆", "蜜桃紅烏龍"},
	{"葡油十五茉", "葡柚十五茉"},
	{"葡又十五茉", "葡柚十五茉"},
	{"清檸桂蜜", "青檸桂蜜"},
	{"青寧桂蜜", "青檸桂蜜"},
	{"清寧桂蜜", "青檸桂蜜"},
	{"金杯密香紅", "金杯蜜香紅"},
	{"插花烏龍", "茶花烏龍"},
	{"插花紅烏龍", "茶花紅烏龍"},
	{"茶花紅烏隆", "茶花紅烏龍"},
	{"茶花烏隆", "茶花烏龍"},
	{"紅玉紅查", "紅玉紅茶"},
	{"烏隆", "烏龍"},
	// 甜度
	{"伴糖", "半糖"},
	{"班糖", "半糖"},
	{"為糖", "微糖"},
	{"威糖", "微糖"},
	// 冰量
	{"趣冰", "去冰"},
	{"為冰", "微冰"},
	{"威冰", "微冰"},
	{"熱引", "熱飲"},
	{"熱印", "熱飲"},
	// 分類
	{"仙奶", "鮮奶"},
	// 加料
	{"燕賣粒", "燕麥粒"},
	{"燕賣", "燕麥"},
	{"紅烏隆茶凍", "紅烏龍茶凍"},
	// 數量
	{"涼杯", "兩杯"},
	{"良杯", "兩杯"},
	{"散杯", "三杯"},
}

// ============================================================
// Layer 2: Pinyin fuzzy matching (fallback for unknown typos)
// ============================================================

// term is a vocabulary entry with pre-computed toneless pinyin.
type term struct {
	text      string   // correct text, e.g. "三窨十五茉"
	py        []string // toneless pinyin, e.g. ["san","xun","shi","wu","mo"]
	runeCount int      // number of runes
}

// vocabulary groups by rune length for efficient sliding window.
// key = rune count of the term.
var vocabByLen map[int][]term

// pinyinArgs is shared config: toneless, no tone marks.
var pinyinArgs pinyin.Args

// similarity threshold: ratio of matching pinyin syllables.
// 0.6 means 60% of syllables must match.
const similarityThreshold = 0.6

func init() {
	pinyinArgs = pinyin.NewArgs()
	pinyinArgs.Style = pinyin.Normal // toneless: "san" not "sān"

	// All vocabulary terms that should be fuzzy-matched.
	vocabWords := []string{
		// 品名
		"三窨十五茉",
		"琥珀鐵觀音",
		"朱槿普洱紅",
		"桂花輕烏龍",
		"米香玉露菁",
		"紅玉紅茶",
		"茶花烏龍",
		"茶花紅烏龍",
		"金杯蜜香紅",
		"蜜桃紅烏龍",
		"葡柚十五茉",
		"青檸桂蜜",
		// 甜度
		"正常糖", "少糖", "半糖", "微糖", "無糖",
		// 冰量
		"正常冰", "少冰", "微冰", "去冰", "熱飲",
		// 加料
		"紅烏龍茶凍", "燕麥粒",
		// 數量
		"兩杯", "三杯", "四杯",
	}

	vocabByLen = make(map[int][]term)
	for _, w := range vocabWords {
		py := toPinyin(w)
		rc := utf8.RuneCountInString(w)
		vocabByLen[rc] = append(vocabByLen[rc], term{
			text:      w,
			py:        py,
			runeCount: rc,
		})
	}
}

// toPinyin converts Chinese text to a slice of toneless pinyin syllables.
func toPinyin(s string) []string {
	result := pinyin.Pinyin(s, pinyinArgs)
	out := make([]string, 0, len(result))
	for _, py := range result {
		if len(py) > 0 {
			out = append(out, py[0])
		}
	}
	return out
}

// pinyinSimilarity returns the ratio of matching syllables (0.0 ~ 1.0).
// Compares position by position; both slices must be the same length.
func pinyinSimilarity(a, b []string) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	match := 0
	for i := range a {
		if a[i] == b[i] {
			match++
		}
	}
	return float64(match) / float64(len(a))
}

// fuzzyReplace scans text with sliding windows and replaces substrings
// whose pinyin is similar to a vocabulary term.
func fuzzyReplace(text string) string {
	runes := []rune(text)
	totalRunes := len(runes)

	// Collect all window sizes we need to check.
	windowSizes := make([]int, 0, len(vocabByLen))
	for size := range vocabByLen {
		windowSizes = append(windowSizes, size)
	}

	// Sort descending so longer matches take priority.
	for i := 0; i < len(windowSizes); i++ {
		for j := i + 1; j < len(windowSizes); j++ {
			if windowSizes[j] > windowSizes[i] {
				windowSizes[i], windowSizes[j] = windowSizes[j], windowSizes[i]
			}
		}
	}

	// Track which rune positions have been replaced (avoid double-replace).
	replaced := make([]bool, totalRunes)

	type replacement struct {
		start, end int
		newText    string
	}
	var reps []replacement

	for _, winSize := range windowSizes {
		terms := vocabByLen[winSize]
		for start := 0; start <= totalRunes-winSize; start++ {
			// Skip if any position in this window was already replaced.
			skip := false
			for k := start; k < start+winSize; k++ {
				if replaced[k] {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			window := string(runes[start : start+winSize])

			// Skip if window is already an exact vocabulary match.
			isExact := false
			for _, t := range terms {
				if window == t.text {
					isExact = true
					break
				}
			}
			if isExact {
				// Mark exact-match positions as protected so fuzzy matches
				// don't overlap and corrupt correct text.
				for k := start; k < start+winSize; k++ {
					replaced[k] = true
				}
				continue
			}

			windowPy := toPinyin(window)
			if len(windowPy) == 0 {
				continue
			}

			var bestMatch *term
			bestSim := 0.0

			for i := range terms {
				t := &terms[i]
				if len(t.py) != len(windowPy) {
					continue
				}
				sim := pinyinSimilarity(windowPy, t.py)
				if sim >= similarityThreshold && sim > bestSim {
					bestSim = sim
					bestMatch = t
				}
			}

			if bestMatch != nil {
				reps = append(reps, replacement{start, start + winSize, bestMatch.text})
				for k := start; k < start+winSize; k++ {
					replaced[k] = true
				}
			}
		}
	}

	// Apply replacements (from end to start to preserve indices).
	for i := len(reps) - 1; i >= 0; i-- {
		r := reps[i]
		runes = append(runes[:r.start], append([]rune(r.newText), runes[r.end:]...)...)
	}

	return string(runes)
}

// ============================================================
// Public API
// ============================================================

// NormalizeSTT corrects speech-to-text errors in two passes:
//  1. Exact replacement table (known typos, O(n) fast)
//  2. Pinyin fuzzy matching (unknown typos, sliding window)
func NormalizeSTT(text string) string {
	// Pass 1: exact replacements
	for _, r := range exactReplacements {
		text = strings.ReplaceAll(text, r.old, r.new)
	}

	// Pass 2: pinyin fuzzy fallback
	text = fuzzyReplace(text)

	return text
}
