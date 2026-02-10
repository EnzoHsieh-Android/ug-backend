package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// STTClient is a client for Qwen3-ASR-Flash API (DashScope)
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

// Qwen3-ASR request types
type asrRequest struct {
	Model string   `json:"model"`
	Input asrInput `json:"input"`
}

type asrInput struct {
	Messages []asrMessage `json:"messages"`
}

type asrMessage struct {
	Role    string       `json:"role"`
	Content []asrContent `json:"content"`
}

type asrContent struct {
	Text  string `json:"text,omitempty"`
	Audio string `json:"audio,omitempty"`
}

// Qwen3-ASR response types
type asrResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}

// Transcribe sends audio data to the Qwen3-ASR API and returns the transcription
func (c *STTClient) Transcribe(audioData []byte, filename string, contentType string) (*TranscriptionResult, error) {
	// Determine MIME type for data URI
	mimeType := inferMIMEType(filename, contentType)

	// Base64 encode audio as data URI
	b64 := base64.StdEncoding.EncodeToString(audioData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	// Menu context for better recognition — guide ASR with all domain terms
	menuContext := `這是茶飲店「No.315 Unique Green」的點單語音辨識。
品項名稱：三窨十五茉、桂花輕烏龍、茶花紅烏龍、茶花烏龍、朱槿普洱紅、米香玉露菁、琥珀鐵觀音、金杯蜜香紅、青檸桂蜜、蜜桃紅烏龍、葡柚十五茉。
分類：研選純茶、牧場直達、UG奶茶、果香四溢、雲頂奶蓋。
甜度：正常糖、少糖、半糖、微糖、無糖。
冰量：正常冰、少冰、微冰、去冰、熱飲。
加料：紅烏龍茶凍、燕麥粒。
尺寸：中杯、大杯。
關鍵詞：窨（xūn）、槿（jǐn）、菁（jīng）、玉露、鐵觀音、烏龍、普洱、茉莉、桂花。`

	reqBody := asrRequest{
		Model: c.model,
		Input: asrInput{
			Messages: []asrMessage{
				{Role: "system", Content: []asrContent{{Text: menuContext}}},
				{Role: "user", Content: []asrContent{{Audio: dataURI}}},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST",
		c.baseURL+"/api/v1/services/aigc/multimodal-generation/generation",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stt request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ASR returned %d: %s", resp.StatusCode, string(respBody))
	}

	var asrResp asrResponse
	if err := json.Unmarshal(respBody, &asrResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(asrResp.Output.Choices) == 0 {
		return nil, fmt.Errorf("ASR returned no choices")
	}

	// Extract text from content array
	var texts []string
	for _, c := range asrResp.Output.Choices[0].Message.Content {
		if c.Text != "" {
			texts = append(texts, c.Text)
		}
	}

	return &TranscriptionResult{
		Text: strings.Join(texts, ""),
	}, nil
}

// inferMIMEType determines the audio MIME type from filename or content type
func inferMIMEType(filename, contentType string) string {
	if contentType != "" && contentType != "application/octet-stream" {
		return contentType
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".wav":
		return "audio/wav"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".webm":
		return "audio/webm"
	case ".ogg", ".opus":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	default:
		return "audio/wav"
	}
}
