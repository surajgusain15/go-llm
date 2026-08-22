package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go-llm/internal/config"
	"go-llm/internal/conversation"
	"go-llm/internal/llm"
	"go-llm/internal/prompts"
	"go-llm/internal/service"
)

func main() {

	cfg := config.Load()

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}

	llmClient := llm.NewOllamaClient(
		httpClient,
		cfg.OllamaBaseURL,
		cfg.Model,
	)

	fmt.Println("LLM Chat")
	fmt.Println("Type 'exit' to quit")

	chatService := service.NewChatService(llmClient)
	conv := conversation.NewWithSystemPrompt(prompts.GolangExpert)
	scanner := bufio.NewScanner(os.Stdin)

	for {

		fmt.Print("\nYou: ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		if input == "exit" {
			break
		}

		fmt.Print("\nAssistant: ")

		stream := chatService.Stream(
			context.Background(),
			conv,
			input,
		)

		for result := range stream {

			if result.Err != nil {
				fmt.Printf("\nError: %v\n", result.Err)
				break
			}

			fmt.Print(result.Chunk.Message.Content)

			if result.Chunk.Done {
				fmt.Println()
			}
		}
	}
}
