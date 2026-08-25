package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

func (c *OllamaClient) Stream(
	ctx context.Context,
	req ChatRequest,
) <-chan StreamResult {

	stream := make(chan StreamResult)

	go func() {

		defer close(stream)

		req.Stream = true

		httpReq, err := c.prepareRequest(
			ctx,
			req,
		)
		if err != nil {

			stream <- StreamResult{
				Err: err,
			}

			return
		}

		httpResp, err := c.doChatRequest(
			httpReq,
		)
		if err != nil {

			stream <- StreamResult{
				Err: err,
			}

			return
		}
		defer httpResp.Body.Close()

		decoder := json.NewDecoder(
			httpResp.Body,
		)

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
