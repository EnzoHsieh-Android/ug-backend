package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xiechengru/ug-backend/model"
)

// Validation maps built from menu (single source of truth)
var (
	validSugar map[string]bool
	validIce   map[string]bool
	validSize  map[string]bool
	validAddon map[string]bool
)

func init() {
	m := GetMenu()

	validSugar = make(map[string]bool)
	for _, s := range m.Options.Sugar {
		validSugar[s] = true
	}

	validIce = make(map[string]bool)
	for _, i := range m.Options.Ice {
		validIce[i] = true
	}

	validSize = make(map[string]bool)
	for _, s := range m.Options.Size {
		validSize[s] = true
	}

	validAddon = make(map[string]bool)
	for _, a := range m.Options.Addon {
		validAddon[a.Name] = true
	}
}

type ValidationResult struct {
	Intent          string
	Orders          []model.Order
	Recommendations []model.Recommendation
	Note            string
	Warnings        []string
}

// stripModelTokens cleans up raw LLM output (Qwen tokens, think tags, markdown fences)
func stripModelTokens(raw string) string {
	raw = strings.TrimSpace(raw)

	// Strip <|im_start|> and <|im_end|> tokens (Qwen format)
	raw = strings.ReplaceAll(raw, "<|im_start|>", "")
	raw = strings.ReplaceAll(raw, "<|im_end|>", "")
	raw = strings.TrimSpace(raw)

	// Strip <think>...</think> tags from model output (handle multiple tags)
	if idx := strings.LastIndex(raw, "</think>"); idx != -1 {
		raw = strings.TrimSpace(raw[idx+len("</think>"):])
	}

	// Strip markdown code fences if present
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 3 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	return raw
}

// ParseIntentResponse parses LLM output and derives intent from array contents.
// Intent is no longer in the model output — it is inferred:
//   - orders non-empty → "order"
//   - recommendations non-empty (no orders) → "recommend"
//   - both non-empty → "order" (mixed)
//   - both empty → "other" (chitchat/unrecognized)
func ParseIntentResponse(raw string) (*ValidationResult, error) {
	raw = stripModelTokens(raw)

	var resp model.IntentResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		// Fallback: try parsing as legacy OrderResponse
		return parseLegacyResponse(raw)
	}

	result := &ValidationResult{}

	// Validate orders if present
	if len(resp.Orders) > 0 {
		orders, warnings := validateOrders(resp.Orders)
		orders = fillDefaults(orders)
		result.Orders = orders
		result.Warnings = warnings
	}

	// Validate recommendations if present
	if len(resp.Recommendations) > 0 {
		recs, recWarnings := validateRecommendations(resp.Recommendations)
		result.Recommendations = recs
		result.Warnings = append(result.Warnings, recWarnings...)
	}

	// Derive intent from contents
	switch {
	case len(result.Orders) > 0:
		result.Intent = "order"
	case len(result.Recommendations) > 0:
		result.Intent = "recommend"
	default:
		result.Intent = "other"
	}

	return result, nil
}

// ParseAndValidate is the legacy parser that only returns orders.
// It delegates to ParseIntentResponse for backward compatibility.
func ParseAndValidate(raw string) (*ValidationResult, error) {
	return ParseIntentResponse(raw)
}

// parseLegacyResponse handles the old format: {"orders":[...]}
func parseLegacyResponse(raw string) (*ValidationResult, error) {
	var resp model.OrderResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON from model: %w", err)
	}

	orders, warnings := validateOrders(resp.Orders)
	return &ValidationResult{
		Intent:   "order",
		Orders:   orders,
		Warnings: warnings,
	}, nil
}

// validateOrders validates a slice of orders against the menu
func validateOrders(orders []model.Order) ([]model.Order, []string) {
	var validated []model.Order
	var allWarnings []string

	for i := range orders {
		o := &orders[i]
		var warnings []string

		// Validate item is on menu (skip not_on_menu items)
		if !o.NotOnMenu {
			if !IsValidMenuItem(o.Item) {
				if matched, ok := FuzzyMatchMenuItem(o.Item); ok {
					original := o.Item
					o.Item = matched
					warnings = append(warnings, fmt.Sprintf("品項「%s」已修正為「%s」", original, matched))
				} else {
					o.NotOnMenu = true
					warnings = append(warnings, fmt.Sprintf("品項「%s」不在菜單上", o.Item))
				}
			}
		}

		// Validate sugar
		if o.Sugar != nil && *o.Sugar != "" && !validSugar[*o.Sugar] {
			warnings = append(warnings, fmt.Sprintf("甜度「%s」不合法，已清除", *o.Sugar))
			o.Sugar = nil
		}

		// Validate ice
		if o.Ice != nil && *o.Ice != "" && !validIce[*o.Ice] {
			warnings = append(warnings, fmt.Sprintf("冰量「%s」不合法，已清除", *o.Ice))
			o.Ice = nil
		}

		// Validate size
		if o.Size != nil && *o.Size != "" && !validSize[*o.Size] {
			warnings = append(warnings, fmt.Sprintf("尺寸「%s」不合法，已清除", *o.Size))
			o.Size = nil
		}

		// Validate and filter addons
		var validAddons []string
		var removedAddons []string
		for _, a := range o.Addon {
			if validAddon[a] {
				validAddons = append(validAddons, a)
			} else {
				removedAddons = append(removedAddons, a)
			}
		}
		if len(removedAddons) > 0 {
			note := fmt.Sprintf("不提供加料：%s", strings.Join(removedAddons, "、"))
			if o.Note != "" {
				o.Note += "；" + note
			} else {
				o.Note = note
			}
			warnings = append(warnings, note)
		}
		o.Addon = validAddons
		if o.Addon == nil {
			o.Addon = []string{}
		}

		// Validate qty
		if o.Qty < 1 {
			o.Qty = 1
			warnings = append(warnings, "數量小於 1，已設為 1")
		}

		allWarnings = append(allWarnings, warnings...)
		validated = append(validated, *o)
	}

	return validated, allWarnings
}

// getDefaultCategory returns the default category for a menu item.
// Priority: pure_tea → rich_milk_tea → first matching category.
func getDefaultCategory(itemName string) string {
	for _, cat := range menu.Categories {
		if cat.ID == "pure_tea" {
			for _, item := range cat.Items {
				if item.Name == itemName {
					return "pure_tea"
				}
			}
		}
	}
	for _, cat := range menu.Categories {
		if cat.ID == "rich_milk_tea" {
			for _, item := range cat.Items {
				if item.Name == itemName {
					return "rich_milk_tea"
				}
			}
		}
	}
	for _, cat := range menu.Categories {
		for _, item := range cat.Items {
			if item.Name == itemName {
				return cat.ID
			}
		}
	}
	return ""
}

// fillDefaults fills in default values for on-menu items where the model output null.
// This handles: category, size, sugar, ice defaults so the model doesn't need to.
func fillDefaults(orders []model.Order) []model.Order {
	for i := range orders {
		o := &orders[i]

		// Skip not-on-menu items
		if o.NotOnMenu || !IsValidMenuItem(o.Item) {
			continue
		}

		// Fill default category
		if o.Category == "" {
			o.Category = getDefaultCategory(o.Item)
		}

		// Fill default size based on category
		// Look up whether item has size in this category
		for _, cat := range menu.Categories {
			if cat.ID == o.Category {
				for _, item := range cat.Items {
					if item.Name == o.Item {
						if !item.HasSize {
							o.Size = nil
						} else if o.Size == nil || *o.Size == "" {
							m := "M"
							o.Size = &m
						}
						break
					}
				}
				break
			}
		}

		// Fill default sugar
		if o.Sugar == nil || *o.Sugar == "" {
			s := "正常糖"
			o.Sugar = &s
		}

		// Fill default ice
		if o.Ice == nil || *o.Ice == "" {
			ic := "正常冰"
			o.Ice = &ic
		}
	}
	return orders
}

// validateRecommendations validates recommended items exist on the menu
func validateRecommendations(recs []model.Recommendation) ([]model.Recommendation, []string) {
	var validated []model.Recommendation
	var warnings []string

	for _, rec := range recs {
		if !IsValidMenuItem(rec.Item) {
			if matched, ok := FuzzyMatchMenuItem(rec.Item); ok {
				original := rec.Item
				rec.Item = matched
				warnings = append(warnings, fmt.Sprintf("推薦品項「%s」已修正為「%s」", original, matched))
			} else {
				warnings = append(warnings, fmt.Sprintf("推薦品項「%s」不在菜單上，已移除", rec.Item))
				continue
			}
		}
		validated = append(validated, rec)
	}

	return validated, warnings
}
