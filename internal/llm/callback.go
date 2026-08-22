package llm

import "context"

// StreamWithCallback consumes the stream and invokes the callback
// for every streamed result.
func StreamWithCallback(
	ctx context.Context,
	client Client,
	req ChatRequest,
	callback func(StreamResult),
) {

	stream := client.Stream(ctx, req)

	for result := range stream {

		callback(result)

		if result.Err != nil {
			return
		}

		if result.Chunk.Done {
			return
		}
	}
}
