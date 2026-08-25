package tools

import (
	"bufio"
	"context"
	"encoding/json"

	"go-llm/internal/llm"
)

type ToolInfo struct {
	Name        string
	Description string
}

type Executor struct {
	registry    *Registry
	middlewares []Middleware
}

func (e *Executor) Use(
	middleware Middleware,
) {

	e.middlewares = append(
		e.middlewares,
		middleware,
	)
}

// func (e *Executor) loadSchemas() {
//
// 	tools := e.registry.List()
//
// 	e.schemas = make(
// 		[]llm.ToolDefinition,
// 		0,
// 		len(tools),
// 	)
//
// 	for _, tool := range tools {
// 		e.schemas = append(
// 			e.schemas,
// 			tool.Schema(),
// 		)
// 	}
// }

func NewExecutor(
	registry *Registry,
) *Executor {

	e := &Executor{
		registry:    registry,
		middlewares: make([]Middleware, 0),
	}

	// e.loadSchemas()

	return e
}

func (e *Executor) List() []ToolInfo {

	tools := e.registry.List()

	result := make([]ToolInfo, 0, len(tools))

	for _, tool := range tools {

		schema := tool.Schema()

		result = append(
			result, ToolInfo{
				Name:        schema.Function.Name,
				Description: schema.Function.Description,
			},
		)
	}

	return result
}

func (e *Executor) Schemas() []llm.ToolDefinition {

	tools := e.registry.List()

	schemas := make(
		[]llm.ToolDefinition,
		0,
		len(tools),
	)

	for _, tool := range tools {
		schemas = append(
			schemas,
			tool.Schema(),
		)
	}

	return schemas
}

func (e *Executor) CollectInput(
	name string,
	reader *bufio.Reader,
) (json.RawMessage, error) {

	tool, err := e.registry.Get(name)
	if err != nil {
		return nil, err
	}

	interactiveTool, ok := tool.(InteractiveTool)
	if !ok {
		return nil, ErrNotInteractive
	}

	return interactiveTool.CollectInput(reader)
}

func (e *Executor) Execute(
	ctx context.Context,
	name string,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	tool, err := e.registry.Get(name)
	if err != nil {
		return nil, err
	}

	handler := func(
		ctx context.Context,
		invocation ToolInvocation,
	) (*llm.ToolResult, error) {

		return tool.Execute(
			ctx,
			invocation.Input,
		)
	}

	for i := len(e.middlewares) - 1; i >= 0; i-- {
		handler = e.middlewares[i](handler)
	}

	return handler(
		ctx,
		ToolInvocation{
			Name:  name,
			Input: input,
		},
	)
}
