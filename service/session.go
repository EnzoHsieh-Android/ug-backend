package service

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xiechengru/ug-backend/model"
)

// nextOrderID generates an incrementing order ID for the session
func nextOrderID(s *model.Session) string {
	s.OrderIDCounter++
	return strconv.Itoa(s.OrderIDCounter)
}

// SessionManager manages voice ordering sessions with in-memory storage
type SessionManager struct {
	sessions map[string]*model.Session
	mu       sync.RWMutex
	llm      *LLMClient
	ttl      time.Duration
	maxSegs  int
}

// NewSessionManager creates a new SessionManager and starts the cleanup goroutine
func NewSessionManager(llm *LLMClient, ttl time.Duration, cleanupInterval time.Duration, maxSegments int) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*model.Session),
		llm:      llm,
		ttl:      ttl,
		maxSegs:  maxSegments,
	}
	go sm.cleanupLoop(cleanupInterval)
	return sm
}

// cleanupLoop periodically removes expired sessions
func (sm *SessionManager) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, s := range sm.sessions {
			// Don't delete completed sessions immediately - keep them for a while
			if s.Status == model.StatusCompleted || s.Status == model.StatusCancelled {
				if now.Sub(s.UpdatedAt) > sm.ttl {
					delete(sm.sessions, id)
				}
			} else if now.Sub(s.UpdatedAt) > sm.ttl {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// CreateSession creates a new session in active status
func (sm *SessionManager) CreateSession() *model.Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	s := &model.Session{
		ID:        uuid.New().String(),
		Status:    model.StatusActive,
		Segments:  []model.Segment{},
		Orders:    []model.Order{},
		Overrides: make(map[string]*model.OrderOverride),
		Warnings:  []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	sm.sessions[s.ID] = s
	return s
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*model.Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return s, nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	delete(sm.sessions, id)
	return nil
}

// AddSegment adds a transcription segment to the session
func (sm *SessionManager) AddSegment(id string, text string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Allow adding segments in active or pending state
	if s.Status != model.StatusActive && s.Status != model.StatusPending {
		return nil, fmt.Errorf("cannot add segment: session status is %s", s.Status)
	}

	if len(s.Segments) >= sm.maxSegs {
		return nil, fmt.Errorf("maximum segments (%d) reached", sm.maxSegs)
	}

	seg := model.Segment{
		Index:     len(s.Segments),
		Text:      text,
		CreatedAt: time.Now(),
	}
	s.Segments = append(s.Segments, seg)
	s.UpdatedAt = time.Now()

	return s, nil
}

// FinalizeSession merges all segments, calls LLM, and applies field-level merge
func (sm *SessionManager) FinalizeSession(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusActive && s.Status != model.StatusPending {
		return nil, fmt.Errorf("cannot finalize: session status is %s", s.Status)
	}

	if len(s.Segments) == 0 {
		return nil, fmt.Errorf("no segments to finalize")
	}

	// Merge all segment texts
	var texts []string
	for _, seg := range s.Segments {
		texts = append(texts, seg.Text)
	}
	mergedText := strings.Join(texts, ";")

	// Apply STT normalization
	normalized := NormalizeSTT(mergedText)
	s.MergedText = normalized // 儲存校正後的文字

	// Log normalization if changed
	if normalized != mergedText {
		log.Printf("[Session %s] 拼音校正: %s → %s", id, mergedText, normalized)
	}

	// Call LLM
	raw, err := sm.llm.Complete(normalized)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}
	log.Printf("[Session %s] LLM 原始輸出: %s", id, raw)

	// Parse and validate (supports intent-based and legacy formats)
	result, err := ParseIntentResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	log.Printf("[Session %s] ParseIntentResponse: intents=%v, %d 筆訂單, %d 筆推薦", id, result.Intents, len(result.Orders), len(result.Recommendations))

	// Apply field-level merge with overrides
	log.Printf("[Session %s] mergeWithOverrides 前: s.Orders=%d, overrides=%d", id, len(s.Orders), len(s.Overrides))
	finalOrders := sm.mergeWithOverrides(s, result.Orders)
	log.Printf("[Session %s] mergeWithOverrides 後: finalOrders=%d", id, len(finalOrders))

	s.Orders = finalOrders

	// Filter recommendations:
	// - First time: always allow
	// - After first recommendation sent: only allow if latest intent is "recommend"
	//   Fallback to keyword check when model doesn't output intents (backward compat)
	if s.SentRecommendations == nil {
		s.SentRecommendations = make(map[string]bool)
	}
	lastIntent := ""
	if len(result.Intents) > 0 {
		lastIntent = result.Intents[len(result.Intents)-1]
	}
	allowRecs := len(s.SentRecommendations) == 0 || lastIntent == "recommend"

	var newRecs []model.Recommendation
	if allowRecs {
		for _, rec := range result.Recommendations {
			newRecs = append(newRecs, rec)
			s.SentRecommendations[rec.Item] = true
		}
	}
	s.Recommendations = newRecs

	s.Intent = result.Intent
	s.Note = result.Note
	s.Warnings = result.Warnings
	s.Status = model.StatusPending
	s.UpdatedAt = time.Now()

	return s, nil
}

// mergeWithOverrides implements the field-level merge algorithm using order ID matching.
// Matching priority: (item + base category) exact match → item-only fallback match.
// Manually deleted items are skipped; manually added items are always preserved.
func (sm *SessionManager) mergeWithOverrides(s *model.Session, llmOrders []model.Order) []model.Order {
	var finalOrders []model.Order

	// Collect matchable overrides from s.Orders (preserving order).
	// Only non-deleted, non-manually-added overrides participate in matching.
	type candidate struct {
		id       string
		override *model.OrderOverride
		used     bool
	}
	var candidates []*candidate
	for _, o := range s.Orders {
		if o.ID == "" {
			continue
		}
		ov, ok := s.Overrides[o.ID]
		if !ok || ov.Deleted || ov.ManuallyAdded {
			continue
		}
		candidates = append(candidates, &candidate{id: o.ID, override: ov})
	}

	// Match each LLM order to an existing override
	for _, llmOrder := range llmOrders {
		var matched *candidate

		// Pass 1: exact match on (item + base category)
		for _, c := range candidates {
			if c.used {
				continue
			}
			if c.override.BaseOrder.Item == llmOrder.Item && c.override.BaseOrder.Category == llmOrder.Category {
				matched = c
				break
			}
		}

		// Pass 2: item-only fallback (handles category override case)
		if matched == nil {
			for _, c := range candidates {
				if c.used {
					continue
				}
				if c.override.BaseOrder.Item == llmOrder.Item {
					matched = c
					break
				}
			}
		}

		if matched != nil {
			matched.used = true
			// Merge fields and preserve original ID
			merged := sm.mergeOrder(matched.override, llmOrder)
			merged.ID = matched.id
			// Update base order for next comparison
			matched.override.BaseOrder = llmOrder
			finalOrders = append(finalOrders, merged)
		} else {
			// Check if this LLM order matches a deleted override — if so, skip it
			isDeleted := false
			for _, ov := range s.Overrides {
				if !ov.Deleted {
					continue
				}
				if ov.BaseOrder.Item == llmOrder.Item {
					isDeleted = true
					break
				}
			}
			if isDeleted {
				continue
			}

			// New item from LLM — assign new ID
			newID := nextOrderID(s)
			llmOrder.ID = newID
			s.Overrides[newID] = &model.OrderOverride{
				OrderID:         newID,
				BaseOrder:       llmOrder,
				ManualOverrides: make(map[string]interface{}),
				Deleted:         false,
				ManuallyAdded:   false,
			}
			finalOrders = append(finalOrders, llmOrder)
		}
	}

	// Add manually added + non-deleted items (preserved across re-finalize)
	for _, o := range s.Orders {
		if o.ID == "" {
			continue
		}
		ov, ok := s.Overrides[o.ID]
		if !ok {
			continue
		}
		if ov.ManuallyAdded && !ov.Deleted {
			order := ov.BaseOrder
			order.ID = o.ID
			applyManualOverrides(&order, ov.ManualOverrides)
			finalOrders = append(finalOrders, order)
		}
	}

	return finalOrders
}

// mergeOrder performs field-level merge between LLM result and manual overrides
func (sm *SessionManager) mergeOrder(override *model.OrderOverride, llmOrder model.Order) model.Order {
	result := llmOrder

	// For each field, check if LLM changed it from base
	// If LLM value == Base value && manual override exists -> use manual override
	// If LLM value != Base value -> use LLM value, clear manual override

	// Category
	if llmOrder.Category != override.BaseOrder.Category {
		delete(override.ManualOverrides, "category")
	} else if val, ok := override.ManualOverrides["category"]; ok {
		if cat, ok := val.(string); ok {
			result.Category = cat
		}
	}

	// Sugar
	llmSugar := ptrToStr(llmOrder.Sugar)
	baseSugar := ptrToStr(override.BaseOrder.Sugar)
	if llmSugar != baseSugar {
		// LLM explicitly changed this field - clear manual override
		delete(override.ManualOverrides, "sugar")
	} else if val, ok := override.ManualOverrides["sugar"]; ok {
		// Use manual override
		if val == nil {
			result.Sugar = nil
		} else if s, ok := val.(string); ok {
			result.Sugar = &s
		}
	}

	// Ice
	llmIce := ptrToStr(llmOrder.Ice)
	baseIce := ptrToStr(override.BaseOrder.Ice)
	if llmIce != baseIce {
		delete(override.ManualOverrides, "ice")
	} else if val, ok := override.ManualOverrides["ice"]; ok {
		if val == nil {
			result.Ice = nil
		} else if s, ok := val.(string); ok {
			result.Ice = &s
		}
	}

	// Size
	llmSize := ptrToStr(llmOrder.Size)
	baseSize := ptrToStr(override.BaseOrder.Size)
	if llmSize != baseSize {
		delete(override.ManualOverrides, "size")
	} else if val, ok := override.ManualOverrides["size"]; ok {
		if val == nil {
			result.Size = nil
		} else if s, ok := val.(string); ok {
			result.Size = &s
		}
	}

	// Addon
	if !sliceEqual(llmOrder.Addon, override.BaseOrder.Addon) {
		delete(override.ManualOverrides, "addon")
	} else if val, ok := override.ManualOverrides["addon"]; ok {
		if addons, ok := val.([]string); ok {
			result.Addon = addons
		}
	}

	// Qty
	if llmOrder.Qty != override.BaseOrder.Qty {
		delete(override.ManualOverrides, "qty")
	} else if val, ok := override.ManualOverrides["qty"]; ok {
		if qty, ok := val.(int); ok {
			result.Qty = qty
		}
	}

	// Note
	if llmOrder.Note != override.BaseOrder.Note {
		delete(override.ManualOverrides, "note")
	} else if val, ok := override.ManualOverrides["note"]; ok {
		if note, ok := val.(string); ok {
			result.Note = note
		}
	}

	return result
}

// UpdateOrders handles manual order modifications (PATCH /orders).
// Uses order ID to track changes. Orders without an ID are treated as manually added.
func (sm *SessionManager) UpdateOrders(id string, newOrders []model.Order) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusPending {
		return nil, fmt.Errorf("cannot update orders: session status is %s", s.Status)
	}

	// Build map of current orders by ID
	currentMap := make(map[string]*model.Order)
	for i := range s.Orders {
		if s.Orders[i].ID != "" {
			currentMap[s.Orders[i].ID] = &s.Orders[i]
		}
	}

	// Build map of new orders by ID (orders without ID are new)
	newByID := make(map[string]*model.Order)
	for i := range newOrders {
		if newOrders[i].ID != "" {
			newByID[newOrders[i].ID] = &newOrders[i]
		}
	}

	// 1. Current orders not in new list -> mark as deleted
	for oid, curOrder := range currentMap {
		if _, exists := newByID[oid]; !exists {
			if override, ok := s.Overrides[oid]; ok {
				override.Deleted = true
			} else {
				s.Overrides[oid] = &model.OrderOverride{
					OrderID:         oid,
					BaseOrder:       *curOrder,
					ManualOverrides: make(map[string]interface{}),
					Deleted:         true,
					ManuallyAdded:   false,
				}
			}
		}
	}

	// All orders deleted → clear overrides for fresh start
	// This prevents deleted overrides from blocking new LLM orders on next finalize
	if len(newOrders) == 0 {
		s.Overrides = make(map[string]*model.OrderOverride)
		s.Orders = nil
		s.UpdatedAt = time.Now()
		return s, nil
	}

	// 2. New orders without ID -> manually added, assign new ID
	for i := range newOrders {
		if newOrders[i].ID == "" {
			newID := nextOrderID(s)
			newOrders[i].ID = newID
			s.Overrides[newID] = &model.OrderOverride{
				OrderID:         newID,
				BaseOrder:       newOrders[i],
				ManualOverrides: make(map[string]interface{}),
				Deleted:         false,
				ManuallyAdded:   true,
			}
		}
	}

	// 3. Orders with ID present in both -> detect field changes
	for i := range newOrders {
		newOrder := &newOrders[i]
		oid := newOrder.ID
		curOrder, exists := currentMap[oid]
		if !exists {
			continue
		}

		override, ok := s.Overrides[oid]
		if !ok {
			override = &model.OrderOverride{
				OrderID:         oid,
				BaseOrder:       *curOrder,
				ManualOverrides: make(map[string]interface{}),
				Deleted:         false,
				ManuallyAdded:   false,
			}
			s.Overrides[oid] = override
		}

		// Un-delete if was deleted
		override.Deleted = false

		// Record field-level changes (including category)
		if newOrder.Category != curOrder.Category {
			override.ManualOverrides["category"] = newOrder.Category
		}

		if ptrToStr(newOrder.Sugar) != ptrToStr(curOrder.Sugar) {
			if newOrder.Sugar == nil {
				override.ManualOverrides["sugar"] = nil
			} else {
				override.ManualOverrides["sugar"] = *newOrder.Sugar
			}
		}

		if ptrToStr(newOrder.Ice) != ptrToStr(curOrder.Ice) {
			if newOrder.Ice == nil {
				override.ManualOverrides["ice"] = nil
			} else {
				override.ManualOverrides["ice"] = *newOrder.Ice
			}
		}

		if ptrToStr(newOrder.Size) != ptrToStr(curOrder.Size) {
			if newOrder.Size == nil {
				override.ManualOverrides["size"] = nil
			} else {
				override.ManualOverrides["size"] = *newOrder.Size
			}
		}

		if !sliceEqual(newOrder.Addon, curOrder.Addon) {
			override.ManualOverrides["addon"] = newOrder.Addon
		}

		if newOrder.Qty != curOrder.Qty {
			override.ManualOverrides["qty"] = newOrder.Qty
		}

		if newOrder.Note != curOrder.Note {
			override.ManualOverrides["note"] = newOrder.Note
		}
	}

	s.Orders = newOrders
	s.UpdatedAt = time.Now()

	return s, nil
}

// Reset clears segments, orders, and overrides, returning to active state
func (sm *SessionManager) Reset(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusPending {
		return nil, fmt.Errorf("cannot reset: session status is %s", s.Status)
	}

	s.Segments = []model.Segment{}
	s.MergedText = ""
	s.Orders = []model.Order{}
	s.Recommendations = nil
	s.SentRecommendations = nil
	s.Intent = ""
	s.Note = ""
	s.Overrides = make(map[string]*model.OrderOverride)
	s.OrderIDCounter = 0
	s.Warnings = []string{}
	s.Status = model.StatusActive
	s.UpdatedAt = time.Now()

	return s, nil
}

// Confirm moves session to checkout state
func (sm *SessionManager) Confirm(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusPending {
		return nil, fmt.Errorf("cannot confirm: session status is %s", s.Status)
	}

	if len(s.Orders) == 0 {
		return nil, fmt.Errorf("cannot confirm: no orders")
	}

	s.Status = model.StatusCheckout
	s.UpdatedAt = time.Now()

	return s, nil
}

// Back moves session from checkout back to pending
func (sm *SessionManager) Back(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusCheckout {
		return nil, fmt.Errorf("cannot go back: session status is %s", s.Status)
	}

	s.Status = model.StatusPending
	s.UpdatedAt = time.Now()

	return s, nil
}

// Complete marks session as completed (payment done)
func (sm *SessionManager) Complete(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status != model.StatusCheckout {
		return nil, fmt.Errorf("cannot complete: session status is %s", s.Status)
	}

	s.Status = model.StatusCompleted
	s.UpdatedAt = time.Now()

	return s, nil
}

// Cancel marks session as cancelled
func (sm *SessionManager) Cancel(id string) (*model.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if s.Status == model.StatusCompleted {
		return nil, fmt.Errorf("cannot cancel: session already completed")
	}

	s.Status = model.StatusCancelled
	s.UpdatedAt = time.Now()

	return s, nil
}

// Helper functions

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func applyManualOverrides(order *model.Order, overrides map[string]interface{}) {
	for field, val := range overrides {
		switch field {
		case "category":
			if cat, ok := val.(string); ok {
				order.Category = cat
			}
		case "sugar":
			if val == nil {
				order.Sugar = nil
			} else if s, ok := val.(string); ok {
				order.Sugar = &s
			}
		case "ice":
			if val == nil {
				order.Ice = nil
			} else if s, ok := val.(string); ok {
				order.Ice = &s
			}
		case "size":
			if val == nil {
				order.Size = nil
			} else if s, ok := val.(string); ok {
				order.Size = &s
			}
		case "addon":
			if addons, ok := val.([]string); ok {
				order.Addon = addons
			}
		case "qty":
			switch v := val.(type) {
			case int:
				order.Qty = v
			case float64:
				order.Qty = int(v)
			}
		case "note":
			if note, ok := val.(string); ok {
				order.Note = note
			}
		}
	}
}

