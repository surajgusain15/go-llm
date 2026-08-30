package llmutil

import (
	"encoding/json"
	"fmt"
)

func ToolResultToString(
	value any,
) (string, error) {

	switch v := value.(type) {

	case string:
		return v, nil

	case []byte:
		return string(v), nil

	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf(
				"marshal tool result: %w",
				err,
			)
		}

		return string(data), nil
	}
}
