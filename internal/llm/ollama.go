package llm

import (
	"context"
	"encoding/json"
	"net/http"
)

type OllamaClient struct {
	httpClient *http.Client
	baseURL    string
	model      string
}

func NewOllamaClient(httpClient *http.Client, baseURL string, model string) *OllamaClient {
	return &OllamaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		model:      model,
	}
}

func (c *OllamaClient) Chat(
	ctx context.Context,
	req ChatRequest,
) (ChatResponse, error) {
	// Use the default model if one wasn't provided.
	if req.Model == "" {
		req.Model = c.model
	}

	httpReq, err := c.newRequest(ctx, req)
	if err != nil {
		return ChatResponse{}, err
	}

	httpResp, err := c.do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	var response ChatResponse

	err = json.NewDecoder(httpResp.Body).Decode(&response)
	if err != nil {
		return ChatResponse{}, err
	}

	return response, nil
}
