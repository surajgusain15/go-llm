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

		for range maxAgentIterations {

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

			restart := false

			for result := range stream {

				if result.Err != nil {

					s.emitLLMRequestFinished(
						start,
						result.Err,
					)

					out <- result
					return
				}

				// Forward tokens to the caller.
				out <- result

				// Buffer assistant text.
				buffer.Write(
					result.Chunk.Message.Content,
				)

				// Handle tool calls.
				if len(result.Chunk.Message.ToolCalls) > 0 {

					toolCall := result.Chunk.Message.ToolCalls[0]

					// Preserve the assistant message containing the tool call.
					conv.AddMessage(
						result.Chunk.Message,
					)

					result, err := s.executeToolCall(
						ctx,
						toolCall,
					)
					if err != nil {

						s.emitLLMRequestFinished(
							start,
							err,
						)

						out <- llm.StreamResult{
							Err: err,
						}
						return
					}

					err = s.appendToolResult(
						conv,
						toolCall,
						result,
					)
					if err != nil {

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

					restart = true
					break
				}

				// Final assistant response.
				if result.Chunk.Done {

					s.emitLLMRequestFinished(
						start,
						nil,
					)

					response := buffer.String()

					conv.AddAssistantMessage(
						response,
					)

					s.core.Observer.OnEvent(
						events.NewAssistantMessage(
							response,
						),
					)

					return
				}
			}

			if restart {

				// Reset the assistant buffer before the next LLM call.
				buffer = &conversation.AssistantBuffer{}

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
