package llm

import (
	"context"
	"encoding/json"
	"net/http"
)

func (c *OllamaClient) prepareRequest(
	ctx context.Context,
	req ChatRequest,
) (*http.Request, error) {

	if req.Model == "" {
		req.Model = c.model
	}

	return c.newRequest(
		ctx,
		req,
	)
}

func (c *OllamaClient) doChatRequest(
	httpReq *http.Request,
) (*http.Response, error) {

	httpResp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {

		defer httpResp.Body.Close()

		var apiErr struct {
			Error string `json:"error"`
		}

		_ = json.NewDecoder(httpResp.Body).Decode(&apiErr)

		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    apiErr.Error,
		}
	}

	return httpResp, nil
}
