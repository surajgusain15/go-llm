package builtin

import (
	"bufio"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type UUIDTool struct{}

func NewUUIDTool() *UUIDTool {
	return &UUIDTool{}
}

func (u *UUIDTool) Name() string {
	return "uuid"
}

func (u *UUIDTool) Description() string {
	return "Generates a UUID."
}

func (u *UUIDTool) ReadInput(
	reader *bufio.Reader,
) (json.RawMessage, error) {
	return nil, nil
}

func (u *UUIDTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (any, error) {

	return uuid.NewString(), nil
}
