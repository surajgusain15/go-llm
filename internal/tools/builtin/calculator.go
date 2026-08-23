package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go-llm/internal/llm"
)

type CalculatorInput struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (c *CalculatorTool) Schema() llm.ToolDefinition {

	return llm.ToolDefinition{
		Type: llm.ToolTypeFunction,
		Function: llm.ToolFunction{
			Name:        "calculator",
			Description: "Adds two numbers.",
			Parameters: llm.ToolParameters{
				Type: "object",
				Required: []string{
					"a",
					"b",
				},
				Properties: map[string]llm.ToolProperty{
					"a": {
						Type:        "number",
						Description: "First number",
					},
					"b": {
						Type:        "number",
						Description: "Second number",
					},
				},
			},
		},
	}
}

func (c *CalculatorTool) CollectInput(
	reader *bufio.Reader,
) (json.RawMessage, error) {

	fmt.Print("First number: ")

	aStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	fmt.Print("Second number: ")

	bStr, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	a, err := strconv.ParseFloat(
		strings.TrimSpace(aStr),
		64,
	)
	if err != nil {
		return nil, err
	}

	b, err := strconv.ParseFloat(
		strings.TrimSpace(bStr),
		64,
	)
	if err != nil {
		return nil, err
	}

	return json.Marshal(
		CalculatorInput{
			A: a,
			B: b,
		},
	)
}

func (c *CalculatorTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	var req CalculatorInput

	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	return &llm.ToolResult{
		Content: req.A + req.B,
	}, nil
}
