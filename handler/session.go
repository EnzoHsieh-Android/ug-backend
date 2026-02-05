package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xiechengru/ug-backend/model"
	"github.com/xiechengru/ug-backend/service"
)

// SessionHandler handles session-related HTTP requests
type SessionHandler struct {
	sessionMgr   *service.SessionManager
	sttClient    *service.STTClient
	maxAudioSize int64 // in bytes
}

// NewSessionHandler creates a new SessionHandler
func NewSessionHandler(sm *service.SessionManager, stt *service.STTClient, maxAudioSizeMB int) *SessionHandler {
	return &SessionHandler{
		sessionMgr:   sm,
		sttClient:    stt,
		maxAudioSize: int64(maxAudioSizeMB) * 1024 * 1024,
	}
}

// CreateSession handles POST /api/session
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s := h.sessionMgr.CreateSession()

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
	})
}

// GetSession handles GET /api/session/{id}
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")
	s, err := h.sessionMgr.GetSession(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID:  s.ID,
		Status:     s.Status,
		MergedText: s.MergedText,
		Orders:     s.Orders,
		Warnings:   s.Warnings,
	})
}

// AddSegment handles POST /api/session/{id}/segment
func (h *SessionHandler) AddSegment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	// Check if session exists
	_, err := h.sessionMgr.GetSession(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: err.Error()})
		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxAudioSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(h.maxAudioSize); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: "failed to parse form: " + err.Error()})
		return
	}

	// Get audio file
	file, header, err := r.FormFile("audio")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: "audio file required"})
		return
	}
	defer file.Close()

	// Read audio data
	audioData, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: "failed to read audio: " + err.Error()})
		return
	}

	log.Printf("[Session %s] 收到音檔: %s, 大小: %.2f KB", id, header.Filename, float64(len(audioData))/1024)

	// Transcribe using Whisper
	result, err := h.sttClient.Transcribe(audioData, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("STT error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: "transcription failed: " + err.Error()})
		return
	}

	log.Printf("[Session %s] Whisper 轉錄結果: %s", id, result.Text)

	// Add segment to session
	s, err := h.sessionMgr.AddSegment(id, result.Text)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SegmentResponse{Error: err.Error()})
		return
	}

	lastIdx := len(s.Segments) - 1
	json.NewEncoder(w).Encode(model.SegmentResponse{
		SegmentIndex:  lastIdx,
		Text:          s.Segments[lastIdx].Text,
		TotalSegments: len(s.Segments),
	})
}

// FinalizeSession handles POST /api/session/{id}/finalize
func (h *SessionHandler) FinalizeSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	log.Printf("[Session %s] 開始分析訂單...", id)

	s, err := h.sessionMgr.FinalizeSession(id)
	if err != nil {
		log.Printf("[Session %s] 分析失敗: %v", id, err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	log.Printf("[Session %s] 合併文字: %s", id, s.MergedText)
	log.Printf("[Session %s] 解析到 %d 個訂單項目", id, len(s.Orders))
	for i, order := range s.Orders {
		log.Printf("[Session %s] 訂單 %d: %s (size=%v, sugar=%v, ice=%v)", id, i+1, order.Item, order.Size, order.Sugar, order.Ice)
	}
	if len(s.Warnings) > 0 {
		log.Printf("[Session %s] 警告: %v", id, s.Warnings)
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID:  s.ID,
		Status:     s.Status,
		MergedText: s.MergedText,
		Orders:     s.Orders,
		Warnings:   s.Warnings,
	})
}

// UpdateOrders handles PATCH /api/session/{id}/orders
func (h *SessionHandler) UpdateOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	var req model.OrdersUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: "invalid request body"})
		return
	}

	s, err := h.sessionMgr.UpdateOrders(id, req.Orders)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
		Orders:    s.Orders,
	})
}

// ResetSession handles POST /api/session/{id}/reset
func (h *SessionHandler) ResetSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	s, err := h.sessionMgr.Reset(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
	})
}

// ConfirmSession handles POST /api/session/{id}/confirm
func (h *SessionHandler) ConfirmSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	s, err := h.sessionMgr.Confirm(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
		Orders:    s.Orders,
	})
}

// BackSession handles POST /api/session/{id}/back
func (h *SessionHandler) BackSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	s, err := h.sessionMgr.Back(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
		Orders:    s.Orders,
	})
}

// CompleteSession handles POST /api/session/{id}/complete
func (h *SessionHandler) CompleteSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	s, err := h.sessionMgr.Complete(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
	})
}

// CancelSession handles POST /api/session/{id}/cancel
func (h *SessionHandler) CancelSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	s, err := h.sessionMgr.Cancel(id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(model.SessionResponse{
		SessionID: s.ID,
		Status:    s.Status,
	})
}

// DeleteSession handles DELETE /api/session/{id}
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := chi.URLParam(r, "id")

	if err := h.sessionMgr.DeleteSession(id); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(model.SessionResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
