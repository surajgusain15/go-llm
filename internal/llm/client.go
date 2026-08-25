package llm

import "context"

type Client interface {
	Chat(
		ctx context.Context,
		req ChatRequest,
	) (ChatResponse, error)

	Stream(
		ctx context.Context,
		req ChatRequest,
	) <-chan StreamResult
}
