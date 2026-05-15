package providers

import "context"

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Content   string `json:"content"`
	TokenIn   int    `json:"token_in"`
	TokenOut  int    `json:"token_out"`
	LatencyMs int    `json:"latency_ms"`
}

type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

