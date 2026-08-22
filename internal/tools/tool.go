package tools

import (
	"context"
	"encoding/json"
)

type Tool interface {
	Name() string

	Description() string

	Execute(
		ctx context.Context,
		input json.RawMessage,
	) (any, error)
}
