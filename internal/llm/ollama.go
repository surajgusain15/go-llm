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

	httpReq, err := c.prepareRequest(
		ctx,
		req,
	)
	if err != nil {
		return ChatResponse{}, err
	}

	httpResp, err := c.doChatRequest(
		httpReq,
	)
	if err != nil {
		return ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	var response ChatResponse

	if err := json.NewDecoder(
		httpResp.Body,
	).Decode(&response); err != nil {

		return ChatResponse{}, err
	}

	return response, nil
}
