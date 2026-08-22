package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"time"
)

type TimeTool struct{}

func NewTimeTool() *TimeTool {
	return &TimeTool{}
}

func (t *TimeTool) Name() string {
	return "current_time"
}

func (t *TimeTool) Description() string {
	return "Returns the current local time."
}

func (t *TimeTool) ReadInput(
	reader *bufio.Reader,
) (json.RawMessage, error) {
	return nil, nil
}

func (t *TimeTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (any, error) {

	return time.Now().Format(time.RFC3339), nil
}
