package model

type Order struct {
	ID        string   `json:"id,omitempty"`
	Item      string   `json:"item"`
	Category  string   `json:"category,omitempty"`
	Size      *string  `json:"size"`
	Ice       *string  `json:"ice"`
	Sugar     *string  `json:"sugar"`
	Addon     []string `json:"addon"`
	Qty       int      `json:"qty"`
	Note      string   `json:"note,omitempty"`
	NotOnMenu bool     `json:"not_on_menu,omitempty"`
}

type OrderResponse struct {
	Orders []Order `json:"orders"`
	Note   string  `json:"note,omitempty"`
}

type OrderRequest struct {
	Text string `json:"text"`
}

// Recommendation represents a menu item recommendation from the model
type Recommendation struct {
	Item     string `json:"item"`
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// Preferences captures the user's taste/scenario preferences for recommendations
type Preferences struct {
	Taste       string `json:"taste,omitempty"`
	Temperature string `json:"temperature,omitempty"`
	Scenario    string `json:"scenario,omitempty"`
	Avoid       string `json:"avoid,omitempty"`
}

// IntentResponse is the raw structure parsed from the LLM output
type IntentResponse struct {
	Intents         []string         `json:"intents,omitempty"`
	Intent          string           `json:"intent"`
	Orders          []Order          `json:"orders,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	Preferences     *Preferences     `json:"preferences,omitempty"`
	Note            string           `json:"note,omitempty"`
}

type APIResponse struct {
	Success         bool             `json:"success"`
	Intent          string           `json:"intent,omitempty"`
	Orders          []Order          `json:"orders,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	Note            string           `json:"note,omitempty"`
	Raw             string           `json:"raw,omitempty"`
	Error           string           `json:"error,omitempty"`
}
