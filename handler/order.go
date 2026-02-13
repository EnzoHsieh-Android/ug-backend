package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/xiechengru/ug-backend/model"
	"github.com/xiechengru/ug-backend/service"
)

type OrderHandler struct {
	llm *service.LLMClient
}

func NewOrderHandler(llm *service.LLMClient) *OrderHandler {
	return &OrderHandler{llm: llm}
}

func (h *OrderHandler) HandleOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req model.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.APIResponse{
			Error: "invalid request body",
		})
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.APIResponse{
			Error: "text is required",
		})
		return
	}

	// Normalize STT typos before sending to LLM
	normalized := service.NormalizeSTT(req.Text)

	// Call LLM
	raw, err := h.llm.Complete(normalized)
	if err != nil {
		log.Printf("LLM error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(model.APIResponse{
			Error: "model unavailable",
		})
		return
	}

	// Parse and validate (supports intent-based and legacy formats)
	result, err := service.ParseIntentResponse(raw)
	if err != nil {
		log.Printf("Validation error: %v, raw: %s", err, raw)
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(model.APIResponse{
			Error: err.Error(),
			Raw:   raw,
		})
		return
	}

	// Combine model note and validation warnings
	var noteParts []string
	if result.Note != "" {
		noteParts = append(noteParts, result.Note)
	}
	if len(result.Warnings) > 0 {
		noteParts = append(noteParts, result.Warnings...)
	}
	note := strings.Join(noteParts, "；")

	json.NewEncoder(w).Encode(model.APIResponse{
		Success:         true,
		Intent:          result.Intent,
		Orders:          result.Orders,
		Recommendations: result.Recommendations,
		Note:            note,
		Raw:             raw,
	})
}
