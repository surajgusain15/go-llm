package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const chatEndpoint = "/api/chat"

func (c *OllamaClient) newRequest(
	ctx context.Context,
	req ChatRequest,
) (*http.Request, error) {

	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf(
			"marshal request: %w",
			err,
		)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+chatEndpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create request: %w",
			err,
		)
	}

	httpReq.Header.Set(
		"Content-Type",
		"application/json",
	)

	return httpReq, nil
}
