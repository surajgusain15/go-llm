package service

import (
	"context"
	"errors"
	"time"

	"go-llm/internal/conversation"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

func (s *ChatService) executeStreamingAgentLoop(
	ctx context.Context,
	conv *conversation.Conversation,
) <-chan llm.StreamResult {

	out := make(chan llm.StreamResult)

	go func() {

		defer close(out)

		buffer := &conversation.AssistantBuffer{}

		for i := 0; i < maxAgentIterations; i++ {

			s.core.Emit(
				events.NewAgentIterationStarted(
					i + 1,
				),
			)

			req := s.buildChatRequest(
				conv,
				true,
			)

			start := time.Now()

			s.emitLLMRequestStarted()

			stream := s.client.Stream(
				ctx,
				req,
			)

			needsAnotherIteration := false

			for chunk := range stream {

				if chunk.Err != nil {

					s.emitLLMRequestFinished(
						start,
						chunk.Err,
					)

					out <- chunk
					return
				}

				// Forward streamed tokens.
				out <- chunk

				// Accumulate assistant text.
				buffer.Write(
					chunk.Chunk.Message.Content,
				)

				// Tool call.
				if len(chunk.Chunk.Message.ToolCalls) > 0 {

					toolCall := chunk.Chunk.Message.ToolCalls[0]

					// Preserve assistant message containing tool call.
					conv.AddMessage(
						chunk.Chunk.Message,
					)

					toolResult, toolErr := s.executeToolCall(
						ctx,
						toolCall,
					)

					executed := ToolExecutionResult{
						Call:   toolCall,
						Result: toolResult,
						Err:    toolErr,
					}

					if err := s.appendToolResult(
						conv,
						executed,
					); err != nil {

						s.emitLLMRequestFinished(
							start,
							err,
						)

						out <- llm.StreamResult{
							Err: err,
						}

						return
					}

					s.emitLLMRequestFinished(
						start,
						nil,
					)

					buffer = &conversation.AssistantBuffer{}

					needsAnotherIteration = true

					break
				}

				// Final assistant response.
				if chunk.Chunk.Done {

					response := buffer.String()

					conv.AddAssistantMessage(
						response,
					)

					s.core.Emit(
						events.NewAssistantMessage(
							response,
						),
					)

					s.emitLLMRequestFinished(
						start,
						nil,
					)

					s.core.Emit(
						events.NewAgentFinished(),
					)

					return
				}
			}

			if needsAnotherIteration {
				continue
			}
		}

		out <- llm.StreamResult{
			Err: errors.New(
				"maximum tool iterations exceeded",
			),
		}
	}()

	return out
}
