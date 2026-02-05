package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// STTClient is a client for OpenAI Whisper API compatible endpoints
type STTClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// TranscriptionResult contains the result of audio transcription
type TranscriptionResult struct {
	Text string `json:"text"`
}

// NewSTTClient creates a new STT client
func NewSTTClient(baseURL, apiKey, model string, timeout time.Duration) *STTClient {
	return &STTClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Transcribe sends audio data to the Whisper API and returns the transcription
func (c *STTClient) Transcribe(audioData []byte, filename string, contentType string) (*TranscriptionResult, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file part
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("write audio data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", c.model); err != nil {
		return nil, fmt.Errorf("write model field: %w", err)
	}

	// Add language field (Traditional Chinese)
	if err := writer.WriteField("language", "zh"); err != nil {
		return nil, fmt.Errorf("write language field: %w", err)
	}

	// Add response format
	if err := writer.WriteField("response_format", "json"); err != nil {
		return nil, fmt.Errorf("write response_format field: %w", err)
	}

	// Add prompt with menu items for better recognition
	// Whisper prompt limit is ~224 tokens, keep it concise
	menuPrompt := "茶飲點單：三窨十五茉、桂花輕烏龍、茶花紅烏龍、朱槿普洱紅、米香玉露菁、茶花烏龍、琥珀鐵觀音、金杯蜜香紅、青檸桂蜜、蜜桃紅烏龍、葡柚十五茉。甜度：正常糖、少糖、半糖、微糖、無糖。冰塊：正常冰、少冰、微冰、去冰、熱飲。加料：紅烏龍茶凍、燕麥粒。牧場直達、UG奶茶、雲頂奶蓋。"
	if err := writer.WriteField("prompt", menuPrompt); err != nil {
		return nil, fmt.Errorf("write prompt field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", c.baseURL+"/v1/audio/transcriptions", &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("whisper returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result TranscriptionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
