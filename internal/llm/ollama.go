package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"redterm/internal/config"
)

type ollamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func newOllama(cfg *config.Config) *ollamaProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &ollamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   cfg.Model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *ollamaProvider) Complete(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"stream": true,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody) //nolint:errcheck
		return "", fmt.Errorf("ollama error %d: %v", resp.StatusCode, errBody)
	}

	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		result.WriteString(chunk.Message.Content)
		if chunk.Done {
			break
		}
	}
	return result.String(), scanner.Err()
}
