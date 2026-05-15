package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aios/internal/providers"
)

type HTTPProvider struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func (p *HTTPProvider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	if p.BaseURL == "" {
		return providers.ChatResponse{}, errors.New("missing_base_url")
	}

	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	payload, err := json.Marshal(map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
	})
	if err != nil {
		return providers.ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return providers.ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return providers.ChatResponse{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return providers.ChatResponse{}, fmt.Errorf("openai_http_error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return providers.ChatResponse{}, err
	}

	if len(parsed.Choices) == 0 {
		return providers.ChatResponse{}, errors.New("openai_empty_choices")
	}

	return providers.ChatResponse{
		Content:   parsed.Choices[0].Message.Content,
		TokenIn:   parsed.Usage.PromptTokens,
		TokenOut:  parsed.Usage.CompletionTokens,
		LatencyMs: int(time.Since(start).Milliseconds()),
	}, nil
}

