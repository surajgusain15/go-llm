package tools

import (
	"errors"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/database"
	"go-llm/internal/tools/builtin"
)

func NewDefaultExecutor(rt *core.Core, db database.Client) *Executor {

	registry := NewRegistry()

	registry.Register(
		builtin.NewCalculatorTool(),
	)

	registry.Register(
		builtin.NewTimeTool(),
	)

	registry.Register(
		builtin.NewUUIDTool(),
	)

	registry.Register(
		builtin.NewDatabaseQueryTool(db),
	)

	registry.Register(
		builtin.NewDatabaseSchemaTool(db),
	)

	executor := NewExecutor(
		registry,
		rt,
		WithToolObservability(),
		WithToolRetry(
			ToolRetryPolicy{
				MaxAttempts: 3,

				ShouldRetry: func(err error) bool {
					return errors.Is(err, ErrToolUnavailable) ||
						errors.Is(err, ErrToolRateLimited) ||
						errors.Is(err, ErrToolTemporary)
				},

				Backoff: ExponentialBackoffWithJitter(
					100*time.Millisecond,
					5*time.Second,
					50*time.Millisecond,
				),

				Budget: &ToolRetryBudgetPolicy{
					PerTool: map[string]int{
						"database_query": 5,
					},
					Global: 20,
				},
			},
		),

		WithMaxToolConcurrency(8),

		WithDefaultToolTimeout(5*time.Second),
	)

	return executor
}
