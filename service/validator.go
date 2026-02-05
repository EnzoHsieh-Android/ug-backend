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
	Orders   []model.Order
	Warnings []string
}

func ParseAndValidate(raw string) (*ValidationResult, error) {
	// Strip special tokens from model output
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
		// Remove first and last lines (```json and ```)
		if len(lines) >= 3 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var resp model.OrderResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON from model: %w", err)
	}

	result := &ValidationResult{}

	for i := range resp.Orders {
		o := &resp.Orders[i]
		var warnings []string

		// Validate item is on menu (skip not_on_menu items)
		if !o.NotOnMenu {
			if !IsValidMenuItem(o.Item) {
				// Try fuzzy matching before marking as not_on_menu
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

		result.Warnings = append(result.Warnings, warnings...)
		result.Orders = append(result.Orders, *o)
	}

	return result, nil
}
