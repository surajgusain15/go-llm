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

	// requestJSON, _ := json.MarshalIndent(req, "", "  ")
	// fmt.Println("REQUEST")
	// fmt.Println(string(requestJSON))

	httpReq, err := c.newRequest(ctx, req)
	if err != nil {
		return ChatResponse{}, err
	}

	httpResp, err := c.do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {

		var apiErr struct {
			Error string `json:"error"`
		}

		_ = json.NewDecoder(httpResp.Body).Decode(&apiErr)

		return ChatResponse{}, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    apiErr.Error,
		}
	}

	var response ChatResponse

	err = json.NewDecoder(httpResp.Body).Decode(&response)
	if err != nil {
		return ChatResponse{}, err
	}

	// responseJSON, _ := json.MarshalIndent(response, "", "  ")
	// fmt.Println("RESPONSE")
	// fmt.Println(string(responseJSON))

	return response, nil
}
