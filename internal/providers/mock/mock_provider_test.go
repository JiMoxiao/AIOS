package mock

import (
	"context"
	"testing"

	"aios/internal/providers"
)

func TestMockProvider_Sequence(t *testing.T) {
	p := &Provider{
		Seq: []Step{
			{Response: providers.ChatResponse{Content: "a", TokenIn: 1, TokenOut: 2, LatencyMs: 3}},
			{Response: providers.ChatResponse{Content: "b"}},
		},
	}

	r1, err := p.Chat(context.Background(), providers.ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Content != "a" {
		t.Fatalf("expected a, got %q", r1.Content)
	}

	r2, err := p.Chat(context.Background(), providers.ChatRequest{Model: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Content != "b" {
		t.Fatalf("expected b, got %q", r2.Content)
	}
}

