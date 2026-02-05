package handler

import (
	"encoding/json"
	"net/http"

	"github.com/xiechengru/ug-backend/service"
)

// GetMenu handles GET /api/menu
func GetMenu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service.GetMenu())
}
