package llm

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

	"redterm/internal/config"
)

type openAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func newOpenAI(cfg *config.Config) *openAIProvider {
	baseURL := cfg.BaseURL
	// Use OpenAI's default endpoint unless an explicit override was provided.
	if baseURL == "" || baseURL == "http://localhost:11434" {
		baseURL = "https://api.openai.com/v1"
	}
	return &openAIProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *openAIProvider) Complete(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"stream": true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		// Try to extract a human-readable message from the JSON error body.
		var errBody struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
			// Gemini wraps errors differently
			Message string `json:"message"`
		}
		msg := strings.TrimSpace(string(raw))
		if json.Unmarshal(raw, &errBody) == nil {
			if errBody.Error.Message != "" {
				msg = errBody.Error.Message
			} else if errBody.Message != "" {
				msg = errBody.Message
			}
		}
		return "", fmt.Errorf("openai error %d: %s", resp.StatusCode, msg)
	}

	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			result.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	return result.String(), scanner.Err()
}
