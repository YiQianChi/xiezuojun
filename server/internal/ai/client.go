package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 对接 ZhipuAI/智谱 或其他 OpenAI-compatible API
type Client struct {
	apiKey    string
	baseURL   string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	return &Client{
		apiKey:    apiKey,
		baseURL:   baseURL,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// StreamMessage 流式消息
type StreamMessage struct {
	Content string
	Error   string
}

// Stream 发起流式请求
func (c *Client) Stream(ctx context.Context, model string, messages []Message) (<-chan StreamMessage, error) {
	payload := map[string]interface{}{
		"model":    model,
		"messages": toAPIMessages(messages),
		"stream":   true,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	ch := make(chan StreamMessage, 10)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		c.readSSE(resp.Body, ch)
	}()

	return ch, nil
}

// Complete 非流式请求
func (c *Client) Complete(ctx context.Context, model string, messages []Message) (string, error) {
	payload := map[string]interface{}{
		"model":    model,
		"messages": toAPIMessages(messages),
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", resp.Status)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty response")
}

func (c *Client) readSSE(r io.Reader, ch chan<- StreamMessage) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large SSE lines
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error.Message != "" {
			ch <- StreamMessage{Error: chunk.Error.Message}
			return
		}
		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				ch <- StreamMessage{Content: content}
			}
		}
	}
}

func toAPIMessages(msgs []Message) []map[string]string {
	out := make([]map[string]string, len(msgs))
	for i, m := range msgs {
		out[i] = map[string]string{
			"role":    m.Role,
			"content": m.Content,
		}
	}
	return out
}
