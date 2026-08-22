package tools

import (
	"bufio"
	"encoding/json"
	"errors"
)

var ErrNotInteractive = errors.New("tool is not interactive")

type InteractiveTool interface {
	Tool

	CollectInput(
		reader *bufio.Reader,
	) (json.RawMessage, error)
}

func (e *Executor) InteractiveTool(
	name string,
) (InteractiveTool, error) {

	tool, err := e.registry.Get(name)
	if err != nil {
		return nil, err
	}

	interactiveTool, ok := tool.(InteractiveTool)
	if !ok {
		return nil, ErrNotInteractive
	}

	return interactiveTool, nil
}
