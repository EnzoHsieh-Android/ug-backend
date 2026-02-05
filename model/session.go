package model

import "time"

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	StatusActive    SessionStatus = "active"    // Recording phase - accumulating segments
	StatusPending   SessionStatus = "pending"   // Awaiting order confirmation
	StatusCheckout  SessionStatus = "checkout"  // In checkout process
	StatusCompleted SessionStatus = "completed" // Payment complete, session finished
	StatusCancelled SessionStatus = "cancelled" // Session cancelled
)

// Segment represents a single voice segment with its transcription
type Segment struct {
	Index     int       `json:"index"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderOverride tracks manual modifications to orders for field-level merging
type OrderOverride struct {
	ItemName        string                 `json:"item_name"`        // Item name as key
	BaseOrder       Order                  `json:"base_order"`       // Last LLM-returned order (baseline)
	ManualOverrides map[string]interface{} `json:"manual_overrides"` // Fields modified manually
	Deleted         bool                   `json:"deleted"`          // Whether item was deleted
	ManuallyAdded   bool                   `json:"manually_added"`   // Whether item was added manually (not from LLM)
}

// Session represents a multi-segment voice ordering session
type Session struct {
	ID         string                    `json:"id"`
	Status     SessionStatus             `json:"status"`
	Segments   []Segment                 `json:"segments"`             // Accumulated transcription segments
	MergedText string                    `json:"merged_text"`          // Combined text from all segments
	Orders     []Order                   `json:"orders"`               // Current orders (after merging)
	Overrides  map[string]*OrderOverride `json:"overrides,omitempty"`  // Field-level override tracking by item name
	Warnings   []string                  `json:"warnings,omitempty"`
	CreatedAt  time.Time                 `json:"created_at"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

// SessionResponse is the API response for session operations
type SessionResponse struct {
	SessionID  string        `json:"session_id,omitempty"`
	Status     SessionStatus `json:"status,omitempty"`
	MergedText string        `json:"merged_text,omitempty"`
	Orders     []Order       `json:"orders,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	Error      string        `json:"error,omitempty"`
}

// SegmentResponse is the API response for adding a segment
type SegmentResponse struct {
	SegmentIndex  int    `json:"segment_index"`
	Text          string `json:"text"`
	TotalSegments int    `json:"total_segments"`
	Error         string `json:"error,omitempty"`
}

// OrdersUpdateRequest is the request body for PATCH /orders
type OrdersUpdateRequest struct {
	Orders []Order `json:"orders"`
}
