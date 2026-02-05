package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xiechengru/ug-backend/config"
	"github.com/xiechengru/ug-backend/handler"
	"github.com/xiechengru/ug-backend/middleware"
	"github.com/xiechengru/ug-backend/service"
)

func main() {
	cfg := config.Load()

	// Initialize services
	llm := service.NewLLMClient(cfg.ModelURL)
	stt := service.NewSTTClient(cfg.WhisperURL, cfg.WhisperAPIKey, cfg.WhisperModel, cfg.WhisperTimeout)
	sessionMgr := service.NewSessionManager(llm, cfg.SessionTTL, cfg.SessionCleanupInterval, cfg.MaxSegmentsPerSession)

	// Initialize handlers
	orderHandler := handler.NewOrderHandler(llm)
	sessionHandler := handler.NewSessionHandler(sessionMgr, stt, cfg.MaxAudioSizeMB)

	r := chi.NewRouter()
	r.Use(middleware.CORS)

	// Health check
	r.Get("/api/health", handler.Health)

	// Menu endpoint
	r.Get("/api/menu", handler.GetMenu)

	// Legacy order endpoint (backward compatible)
	r.Post("/api/order", orderHandler.HandleOrder)

	// Session endpoints
	r.Route("/api/session", func(r chi.Router) {
		r.Post("/", sessionHandler.CreateSession)
		r.Get("/{id}", sessionHandler.GetSession)
		r.Delete("/{id}", sessionHandler.DeleteSession)
		r.Post("/{id}/segment", sessionHandler.AddSegment)
		r.Post("/{id}/finalize", sessionHandler.FinalizeSession)
		r.Patch("/{id}/orders", sessionHandler.UpdateOrders)
		r.Post("/{id}/reset", sessionHandler.ResetSession)
		r.Post("/{id}/confirm", sessionHandler.ConfirmSession)
		r.Post("/{id}/back", sessionHandler.BackSession)
		r.Post("/{id}/complete", sessionHandler.CompleteSession)
		r.Post("/{id}/cancel", sessionHandler.CancelSession)
	})

	log.Printf("Starting server on :%s (model: %s, whisper: %s)", cfg.Port, cfg.ModelURL, cfg.WhisperURL)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
