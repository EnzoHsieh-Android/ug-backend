package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port string

	// LLM Model
	ModelURL string

	// Whisper STT
	WhisperURL    string
	WhisperAPIKey string
	WhisperModel  string
	WhisperTimeout time.Duration

	// Session
	SessionTTL             time.Duration
	SessionCleanupInterval time.Duration
	MaxSegmentsPerSession  int
	MaxAudioSizeMB         int
}

func Load() *Config {
	return &Config{
		Port:     getEnv("PORT", "3000"),
		ModelURL: getEnv("MODEL_URL", "http://localhost:8080"),

		WhisperURL:     getEnv("WHISPER_URL", "http://localhost:9000"),
		WhisperAPIKey:  getEnv("WHISPER_API_KEY", ""),
		WhisperModel:   getEnv("WHISPER_MODEL", "whisper-1"),
		WhisperTimeout: getDuration("WHISPER_TIMEOUT", 30*time.Second),

		SessionTTL:             getDuration("SESSION_TTL", 30*time.Minute),
		SessionCleanupInterval: getDuration("SESSION_CLEANUP_INTERVAL", 5*time.Minute),
		MaxSegmentsPerSession:  getInt("MAX_SEGMENTS_PER_SESSION", 20),
		MaxAudioSizeMB:         getInt("MAX_AUDIO_SIZE_MB", 25),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
