package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *OllamaClient) Stream(
	ctx context.Context,
	req ChatRequest,
) <-chan StreamResult {

	stream := make(chan StreamResult)

	go func() {
		defer close(stream)

		// Use the default model if none was provided.
		if req.Model == "" {
			req.Model = c.model
		}

		// Streaming must be enabled.
		req.Stream = true

		httpReq, err := c.newRequest(ctx, req)
		if err != nil {
			stream <- StreamResult{Err: err}
			return
		}

		httpResp, err := c.do(httpReq)
		if err != nil {
			stream <- StreamResult{Err: err}
			return
		}

		// Validate response.
		if httpResp.StatusCode != http.StatusOK {

			responseBody, _ := io.ReadAll(httpResp.Body)

			stream <- StreamResult{
				Err: fmt.Errorf(
					"ollama returned status %d: %s",
					httpResp.StatusCode,
					string(responseBody),
				),
			}

			return
		}

		decoder := json.NewDecoder(httpResp.Body)

		for {

			var response ChatResponse

			if err := decoder.Decode(&response); err != nil {

				if err == io.EOF {
					return
				}

				stream <- StreamResult{
					Err: fmt.Errorf(
						"decode stream response: %w",
						err,
					),
				}

				return
			}

			stream <- StreamResult{
				Chunk: StreamChunk{
					Message: response.Message,
					Done:    response.Done,
				},
			}

			if response.Done {
				return
			}
		}
	}()

	return stream
}
