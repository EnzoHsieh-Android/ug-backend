package model

type Order struct {
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

type APIResponse struct {
	Success bool     `json:"success"`
	Orders  []Order  `json:"orders,omitempty"`
	Note    string   `json:"note,omitempty"`
	Raw     string   `json:"raw,omitempty"`
	Error   string   `json:"error,omitempty"`
}
