package llm

type StreamMessageType string

const (
	StreamMessageToken StreamMessageType = "token"

	StreamMessageToolCall StreamMessageType = "tool_call"

	StreamMessageCompleted StreamMessageType = "completed"

	StreamMessageError StreamMessageType = "error"
)

type StreamMessage struct {
	Type StreamMessageType

	Token string

	ToolCall *ToolCall

	Err error
}

type StreamChunk struct {
	Message Message

	Done bool

	DoneReason string
}

type StreamResult struct {
	Chunk StreamChunk

	Err error
}
