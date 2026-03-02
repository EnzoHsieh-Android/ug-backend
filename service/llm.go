package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const systemPrompt = `你是 No.315 Unique Green 的點單助手。根據顧客的自然語言輸出 JSON。

菜單分類與品項描述：
【研選純茶 pure_tea】單一尺寸
- 三窨十五茉：三次窨花茉莉綠茶，花香清雅帶甘甜
- 桂花輕烏龍：桂花與輕發酵烏龍，花果蜜韻輕柔
- 茶花紅烏龍：重焙火紅烏龍，天然花香與熟果甜韻
- 朱槿普洱紅：洛神花搭普洱紅茶，微酸回甘解膩
- 米香玉露菁：日式玉露菁帶米穀香，鮮爽細膩

【牧場直達 fresh_milk_tea】M/L
- 三窨十五茉(65/75)：茉莉鮮奶，清香滑順
- 桂花輕烏龍(65/75)：桂花烏龍鮮奶，蜜韻奶香溫潤
- 茶花紅烏龍(65/75)：烏龍鮮奶，茶奶平衡圓潤
- 琥珀鐵觀音(65/75)：重焙鐵觀音鮮奶，焙火香濃郁
- 金杯蜜香紅(60/70)：蜜香紅茶鮮奶，天然蜜甜
- 朱槿普洱紅(60/70)：洛神普洱鮮奶，微酸風味獨特
- 米香玉露菁(60/70)：玉露菁鮮奶，穀物奶香療癒

【UG奶茶 rich_milk_tea】M/L
- 三窨十五茉(65/75)、桂花輕烏龍(65/75)、茶花紅烏龍(65/75)
- 琥珀鐵觀音(65/75)、金杯蜜香紅(60/70)、朱槿普洱紅(60/70)、米香玉露菁(60/70)

【果香四溢 fruit_tea】僅冷飲，單一尺寸
- 青檸桂蜜(65)：青檸搭桂花蜜，酸甜清爽消暑
- 蜜桃紅烏龍(75)：水蜜桃果香與紅烏龍交織，甜蜜芬芳
- 葡柚十五茉(80)：葡萄柚酸香配茉莉，清新微苦大人味

【雲頂奶蓋 cloud_velvet】單一尺寸
- 桂花輕烏龍(95)：桂花烏龍覆綿密奶蓋，鹹甜交融
- 茶花紅烏龍(95)：紅烏龍焙火韻搭奶蓋，茶香奶鹹平衡

加料：紅烏龍茶凍(+15)、燕麥粒(+15)。
甜度：全糖/五分糖/三分糖/一分糖/無糖（顧客若說正常糖/半糖/少糖/微糖亦照錄）。冰量：正常冰/少冰/微冰/去冰/熱飲。
未指定的甜度、冰量、尺寸設為 null。未指定分類時 category 設 null。

intents 規則：
- intents 陣列，每個元素對應一段輸入（以分號分隔），值為 order/recommend/modify/other
- order：點餐或新增品項
- recommend：請求推薦（含口味偏好、場景描述等隱含推薦）
- modify：修改或取消既有品項
- other：閒聊或無法辨識
- 單段輸入時 intents 只有一個元素

輸出格式（必須為合法 JSON）：
{"intents":[],"orders":[],"preferences":{},"recommendations":[]}
- 顧客點餐：填入 orders，每筆格式 {"item":"品名","category":"分類","size":"M/L/null","ice":"冰量/null","sugar":"甜度/null","addon":[],"qty":1,"note":""}
- 顧客想要推薦：填入 preferences（口味/溫度/場景/排除，無明確偏好時為空物件）與 recommendations（僅推薦 1 項），格式 {"item":"品名","category":"分類","reason":"推薦理由"}
- 點餐同時想推薦：orders 和 recommendations 同時填入，preferences 一併填入
- 閒聊或無法辨識：三個欄位皆為空
- 若 intents 中有 recommend，recommendations 必須填入推薦結果（後端決定是否顯示）`

type LLMClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewLLMClient(baseURL string) *LLMClient {
	return &LLMClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
	}
}

// completionRequest uses /v1/completions with manually constructed prompt.
// This bypasses vLLM's chat template and matches RunPod eval format exactly.
type completionRequest struct {
	Model       string            `json:"model"`
	Prompt      string            `json:"prompt"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	Stop        []string          `json:"stop"`
	LogitBias   map[string]int    `json:"logit_bias,omitempty"`
}

type completionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

func (c *LLMClient) Complete(userText string) (string, error) {
	if strings.TrimSpace(userText) == "" {
		return "", fmt.Errorf("empty input text")
	}

	// Construct prompt in Qwen3 chat format with /no_think (matches RunPod eval)
	prompt := "<|im_start|>system\n" + systemPrompt + "<|im_end|>\n" +
		"<|im_start|>user\n" + userText + "<|im_end|>\n" +
		"<|im_start|>assistant\n/no_think\n"

	reqBody := completionRequest{
		Model:       "ug",
		Prompt:      prompt,
		Temperature: 0,
		MaxTokens:   256,
		Stop:        []string{"<|im_end|>"},
		LogitBias:   map[string]int{"151667": -100},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/v1/completions",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(respBody))
	}

	var compResp completionResponse
	if err := json.Unmarshal(respBody, &compResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(compResp.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return compResp.Choices[0].Text, nil
}
