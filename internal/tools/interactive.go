package tools

import (
	"bufio"
	"encoding/json"
	"errors"
)

var ErrNotInteractive = errors.New(
	"tool is not interactive",
)
var ErrToolNotFound = errors.New(
	"tool not found",
)

type InteractiveTool interface {
	Tool

	CollectInput(
		reader *bufio.Reader,
	) (json.RawMessage, error)
}

func (e *Executor) InteractiveTool(
	name string,
) (InteractiveTool, error) {

	tool, ok := e.registry.Get(name)
	if !ok {
		return nil, ErrToolNotFound
	}

	interactiveTool, ok := tool.(InteractiveTool)
	if !ok {
		return nil, ErrNotInteractive
	}

	return interactiveTool, nil
}
