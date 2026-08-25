package tools

import (
	"context"
	"log"
	"time"

	"go-llm/internal/llm"
)

func Logging(
	next Handler,
) Handler {

	return func(
		ctx context.Context,
		invocation ToolInvocation,
	) (*llm.ToolResult, error) {

		start := time.Now()

		result, err := next(
			ctx,
			invocation,
		)

		duration := time.Since(start)

		if err != nil {
			log.Printf(
				"[Tool] %s failed after %s: %v",
				invocation.Name,
				duration,
				err,
			)
		} else {
			log.Printf(
				"[Tool] %s completed in %s",
				invocation.Name,
				duration,
			)
		}

		return result, err
	}
}
