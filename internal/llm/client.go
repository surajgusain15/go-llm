package llm

import "context"

type Client interface {
	// Chat performs a normal blocking request.
	Chat(
		ctx context.Context,
		req ChatRequest,
	) (ChatResponse, error)

	// Stream streams the response incrementally.
	Stream(
		ctx context.Context,
		req ChatRequest,
	) <-chan StreamResult
}
