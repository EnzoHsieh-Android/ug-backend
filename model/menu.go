package model

// MenuItem represents a single menu item
type MenuItem struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	PriceM      int      `json:"price_m,omitempty"`      // M size price (0 = not available)
	PriceL      int      `json:"price_l,omitempty"`      // L size price (0 = not available)
	HasSize     bool     `json:"has_size"`               // Whether item has M/L options
	IceOptions  []string `json:"ice_options,omitempty"`  // Available ice options (nil = all)
	ColdOnly    bool     `json:"cold_only,omitempty"`    // Only cold drinks available
}

// Addon represents an add-on option
type Addon struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// MenuCategory represents a category of menu items
type MenuCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Items []MenuItem `json:"items"`
}

// MenuOptions contains all customization options
type MenuOptions struct {
	Sugar []string `json:"sugar"`
	Ice   []string `json:"ice"`
	Size  []string `json:"size"`
	Addon []Addon  `json:"addon"`
}

// Menu represents the complete menu
type Menu struct {
	Categories []MenuCategory `json:"categories"`
	Options    MenuOptions    `json:"options"`
}
