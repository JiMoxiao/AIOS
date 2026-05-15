package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aios/internal/providers"
)

func TestHTTPProvider_Handles5xx(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	t.Cleanup(s.Close)

	p := &HTTPProvider{
		BaseURL: s.URL,
		Client:  &http.Client{Timeout: 2 * time.Second},
	}

	_, err := p.Chat(context.Background(), providers.ChatRequest{
		Model:    "x",
		Messages: []providers.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestHTTPProvider_Handles429(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate_limited"}`))
	}))
	t.Cleanup(s.Close)

	p := &HTTPProvider{
		BaseURL: s.URL,
		Client:  &http.Client{Timeout: 2 * time.Second},
	}

	_, err := p.Chat(context.Background(), providers.ChatRequest{
		Model:    "x",
		Messages: []providers.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestHTTPProvider_Timeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`))
	}))
	t.Cleanup(s.Close)

	p := &HTTPProvider{
		BaseURL: s.URL,
		Client:  &http.Client{Timeout: 10 * time.Millisecond},
	}

	_, err := p.Chat(context.Background(), providers.ChatRequest{
		Model:    "x",
		Messages: []providers.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestHTTPProvider_Success(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`))
	}))
	t.Cleanup(s.Close)

	p := &HTTPProvider{
		BaseURL: s.URL,
		Client:  &http.Client{Timeout: 2 * time.Second},
	}

	resp, err := p.Chat(context.Background(), providers.ChatRequest{
		Model:    "x",
		Messages: []providers.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" || resp.TokenIn != 1 || resp.TokenOut != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

