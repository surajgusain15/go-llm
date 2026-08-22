package tools

import "fmt"

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {

	r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, error) {

	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf(
			"tool %q not found",
			name,
		)
	}

	return tool, nil
}

func (r *Registry) List() []Tool {

	list := make([]Tool, 0, len(r.tools))

	for _, tool := range r.tools {
		list = append(list, tool)
	}

	return list
}
