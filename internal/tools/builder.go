package tools

import (
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
		WithDefaultToolTimeout(5*time.Second),
		WithToolTimeout(
			"database_query",
			10*time.Second,
		),
		WithToolTimeout(
			"database_schema",
			2*time.Second,
		),
	)

	return executor
}
