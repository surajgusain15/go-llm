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

		for i := range maxAgentIterations {

			s.core.Emit(
				events.NewAgentIterationStarted(i + 1),
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

			buffer := &conversation.AssistantBuffer{}
			var toolCalls []llm.ToolCall

			streamDone := false

			for chunk := range stream {

				if chunk.Err != nil {

					s.emitLLMRequestFinished(
						start,
						chunk.Err,
					)

					out <- chunk
					return
				}

				// Forward the streamed chunk to the caller.
				out <- chunk

				// Accumulate assistant content.
				buffer.Write(
					chunk.Chunk.Message.Content,
				)

				// Accumulate all tool calls from the response.
				if len(chunk.Chunk.Message.ToolCalls) > 0 {
					toolCalls = append(
						toolCalls,
						chunk.Chunk.Message.ToolCalls...,
					)
				}

				if chunk.Chunk.Done {
					streamDone = true
					break
				}
			}

			if !streamDone {

				err := errors.New(
					"LLM stream ended before completion",
				)

				s.emitLLMRequestFinished(
					start,
					err,
				)

				out <- llm.StreamResult{
					Err: err,
				}

				return
			}

			// Normal assistant response.
			if len(toolCalls) == 0 {

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

			// Preserve the assistant message that requested
			// the tools.
			conv.AddMessage(
				llm.Message{
					Role:      llm.RoleAssistant,
					Content:   buffer.String(),
					ToolCalls: toolCalls,
				},
			)

			// Execute all tool calls using the same concurrent
			// execution path as the non-streaming agent.
			results, err := s.executeTools(
				ctx,
				toolCalls,
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

			// Add tool results to the conversation.
			for _, executed := range results {

				if err := s.appendToolResult(
					conv,
					executed.Call,
					executed.Result,
					executed.Err,
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
			}

			s.emitLLMRequestFinished(
				start,
				nil,
			)

			// Continue to the next agent iteration.
		}

		out <- llm.StreamResult{
			Err: errors.New(
				"maximum agent iterations exceeded",
			),
		}
	}()

	return out
}
