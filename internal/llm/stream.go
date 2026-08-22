package llm

// StreamChunk represents a single piece of data received while
// streaming a response from an LLM.
type StreamChunk struct {
	Message Message
	Done    bool
}

// StreamResult represents either a streamed chunk or an error.
type StreamResult struct {
	Chunk StreamChunk
	Err   error
}
