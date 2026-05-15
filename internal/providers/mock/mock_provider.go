package mock

import (
	"context"
	"errors"
	"sync"
	"time"

	"aios/internal/providers"
)

type Provider struct {
	mu   sync.Mutex
	idx  int
	Seq  []Step
	Fall Step
}

type Step struct {
	Response providers.ChatResponse
	Error    error
	Delay    time.Duration
}

func (p *Provider) Chat(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	step := p.next()
	if step.Delay > 0 {
		t := time.NewTimer(step.Delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return providers.ChatResponse{}, ctx.Err()
		case <-t.C:
		}
	}
	if step.Error != nil {
		return providers.ChatResponse{}, step.Error
	}
	return step.Response, nil
}

func (p *Provider) next() Step {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.Seq) == 0 {
		if p.Fall.Error != nil || p.Fall.Response.Content != "" || p.Fall.Delay > 0 {
			return p.Fall
		}
		return Step{Error: errors.New("mock_no_steps")}
	}

	if p.idx >= len(p.Seq) {
		return p.Seq[len(p.Seq)-1]
	}

	s := p.Seq[p.idx]
	p.idx++
	return s
}

