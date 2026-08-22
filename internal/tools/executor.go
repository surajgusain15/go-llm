package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
)

type ToolInfo struct {
	Name        string
	Description string
}

type Executor struct {
	registry *Registry
}

func NewExecutor(
	registry *Registry,
) *Executor {

	return &Executor{
		registry: registry,
	}
}

// List returns metadata for all registered tools.
func (e *Executor) List() []ToolInfo {

	tools := e.registry.List()

	result := make([]ToolInfo, 0, len(tools))

	for _, tool := range tools {

		result = append(
			result, ToolInfo{
				Name:        tool.Name(),
				Description: tool.Description(),
			},
		)
	}

	return result
}

// CollectInput collects interactive input if the tool supports it.
func (e *Executor) CollectInput(
	name string,
	reader *bufio.Reader,
) (json.RawMessage, error) {

	fmt.Println("CollectInput called")

	tool, err := e.registry.Get(name)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Tool Type: %T\n", tool)

	interactiveTool, ok := tool.(InteractiveTool)

	fmt.Println("Interactive:", ok)

	if !ok {
		return nil, ErrToolNotInteractive
	}

	return interactiveTool.CollectInput(reader)
}

// Execute executes a tool.
func (e *Executor) Execute(
	ctx context.Context,
	name string,
	input json.RawMessage,
) (any, error) {

	tool, err := e.registry.Get(name)
	if err != nil {
		return nil, err
	}

	return tool.Execute(
		ctx,
		input,
	)
}
