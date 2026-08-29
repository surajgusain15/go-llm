package tools

import (
	"go-llm/internal/llm"
)

type Registry struct {
	tools map[string]Tool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		order: make([]string, 0),
	}
}

func (r *Registry) Register(tool Tool) {

	schema := tool.Schema()
	name := schema.Function.Name

	// Prevent duplicate registration.
	if _, exists := r.tools[name]; exists {
		return
	}

	r.tools[name] = tool
	r.order = append(
		r.order,
		name,
	)
}

func (r *Registry) Get(
	name string,
) (Tool, bool) {

	tool, ok := r.tools[name]

	return tool, ok
}

func (r *Registry) Schemas() []llm.ToolDefinition {

	schemas := make(
		[]llm.ToolDefinition,
		0,
		len(r.order),
	)

	for _, name := range r.order {

		tool := r.tools[name]

		schemas = append(
			schemas,
			tool.Schema(),
		)
	}

	return schemas
}

func (r *Registry) Tools() []Tool {

	result := make(
		[]Tool,
		0,
		len(r.order),
	)

	for _, name := range r.order {
		result = append(
			result,
			r.tools[name],
		)
	}

	return result
}
