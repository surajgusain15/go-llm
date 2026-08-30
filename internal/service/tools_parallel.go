package service

import (
	"context"

	"golang.org/x/sync/errgroup"

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

	group, groupCtx := errgroup.WithContext(ctx)

	for i, call := range calls {
		i := i
		call := call

		group.Go(
			func() error {

				result, err := s.executeToolCall(
					groupCtx,
					call,
				)

				results[i] = ToolExecutionResult{
					Call:   call,
					Result: result,
					Err:    err,
				}

				// Individual tool failures are represented in
				// ToolExecutionResult and should not abort the batch.
				return nil
			},
		)
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
