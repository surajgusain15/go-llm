package conversation

import "strings"

type AssistantBuffer struct {
	builder strings.Builder
}

func (b *AssistantBuffer) Write(
	content string,
) {
	b.builder.WriteString(content)
}

func (b *AssistantBuffer) String() string {
	return b.builder.String()
}
