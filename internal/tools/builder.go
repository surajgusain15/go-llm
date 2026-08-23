package tools

import "go-llm/internal/tools/builtin"

func NewDefaultExecutor() *Executor {

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

	return NewExecutor(registry)
}
