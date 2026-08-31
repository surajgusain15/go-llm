package tools

import (
	"context"
	"sync"

	"go-llm/internal/llm"
)

type ToolBulkheadPolicy struct {
	PerTool map[string]int
}

type toolBulkhead struct {
	mu sync.Mutex

	slots map[string]chan struct{}
}

func ToolBulkhead(
	policy ToolBulkheadPolicy,
) ToolMiddleware {

	bulkhead := &toolBulkhead{
		slots: make(map[string]chan struct{}),
	}

	for name, limit := range policy.PerTool {
		if limit <= 0 {
			continue
		}

		bulkhead.slots[name] = make(chan struct{}, limit)
	}

	return func(next Handler) Handler {
		return func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			slot := bulkhead.slot(invocation.Name)

			if slot == nil {
				return next(
					ctx,
					invocation,
				)
			}

			select {
			case slot <- struct{}{}:
				defer func() {
					<-slot
				}()

			case <-ctx.Done():
				return nil, ctx.Err()
			}

			return next(
				ctx,
				invocation,
			)
		}
	}
}

func (b *toolBulkhead) slot(
	name string,
) chan struct{} {

	b.mu.Lock()
	defer b.mu.Unlock()

	return b.slots[name]
}
