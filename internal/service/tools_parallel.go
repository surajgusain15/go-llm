package service

import (
	"context"
	"sync"
	"time"

	"go-llm/internal/llm"
)

func (s *ChatService) executeTools(
	ctx context.Context,
	calls []llm.ToolCall,
) ([]ToolExecutionResult, error) {

	if len(calls) == 0 {
		return nil, nil
	}

	results := make([]ToolExecutionResult, len(calls))

	var wg sync.WaitGroup

	wg.Add(len(calls))

	for i, call := range calls {

		i := i
		call := call

		go func() {
			defer wg.Done()
			start := time.Now()

			result, err := s.executeToolCall(
				ctx,
				call,
			)

			results[i] = ToolExecutionResult{
				Call:     call,
				Result:   result,
				Err:      err,
				Duration: time.Since(start),
			}
		}()
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
