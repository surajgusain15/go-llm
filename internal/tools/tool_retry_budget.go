package tools

import (
	"context"
	"sync"
)

type ToolRetryBudgetPolicy struct {
	PerTool map[string]int
	Global  int
}

type toolRetryBudget struct {
	mu sync.Mutex

	perTool map[string]chan struct{}
	global  chan struct{}
}

func newToolRetryBudget(
	policy ToolRetryBudgetPolicy,
) *toolRetryBudget {
	budget := &toolRetryBudget{
		perTool: make(map[string]chan struct{}),
	}

	for name, limit := range policy.PerTool {
		if limit <= 0 {
			continue
		}

		budget.perTool[name] = make(chan struct{}, limit)
	}

	if policy.Global > 0 {
		budget.global = make(chan struct{}, policy.Global)
	}

	return budget
}

func (b *toolRetryBudget) acquire(
	ctx context.Context,
	name string,
) (func(), error) {
	b.mu.Lock()

	perTool := b.perTool[name]
	global := b.global

	b.mu.Unlock()

	if perTool != nil {
		select {
		case perTool <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if global != nil {
		select {
		case global <- struct{}{}:
		case <-ctx.Done():
			if perTool != nil {
				<-perTool
			}

			return nil, ctx.Err()
		}
	}

	return func() {
		if global != nil {
			<-global
		}

		if perTool != nil {
			<-perTool
		}
	}, nil
}
