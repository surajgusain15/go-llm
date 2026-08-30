package tools

import (
	"encoding/json"
)

type ToolInvocation struct {
	Name  string
	Input json.RawMessage
}
