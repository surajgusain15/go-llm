package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go-llm/internal/tools"
)

type CalculatorInput struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}

type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (c *CalculatorTool) Name() string {
	return "calculator"
}

func (c *CalculatorTool) Description() string {
	return "Adds two numbers."
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

	req := CalculatorInput{
		A: a,
		B: b,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CalculatorTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (any, error) {

	var req CalculatorInput

	if err := json.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	return req.A + req.B, nil
}

var _ tools.Tool = (*CalculatorTool)(nil)
var _ tools.InteractiveTool = (*CalculatorTool)(nil)
