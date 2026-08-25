package tools

import (
	"go-llm/internal/core"
	"go-llm/internal/tools/builtin"
)

func NewDefaultExecutor(rt *core.Core) *Executor {

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

	executor := NewExecutor(
		registry,
		rt,
	)

	return executor
}
