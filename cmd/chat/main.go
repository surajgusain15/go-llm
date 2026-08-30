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
	"go-llm/internal/core"
	"go-llm/internal/database"
	"go-llm/internal/events"
	"go-llm/internal/llm"
	"go-llm/internal/prompts"
	"go-llm/internal/service"
	"go-llm/internal/tools"
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
	rt := core.New(events.NewCLIObserver(events.LogLevelInfo))

	dbConfig := database.MySQLConfig{
		DSN:              cfg.MySQLDSN,
		QueryTimeout:     cfg.MySQLQueryTimeout,
		MaxRows:          cfg.MySQLMaxRows,
		MaxResultBytes:   cfg.MySQLMaxResultBytes,
		SchemaTTL:        cfg.MySQLSchemaCacheTTL,
		MaxJoins:         cfg.MySQLMaxJoins,
		MaxUnionBranches: cfg.MySQLMaxUnionBranches,
		MaxSubqueryDepth: cfg.MySQLMaxSubqueryDepth,
	}

	db, err := database.NewMySQLClient(
		dbConfig,
		rt,
	)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		panic(err)
	}

	executor := tools.NewDefaultExecutor(
		rt,
		db,
	)

	fmt.Println("LLM Chat")
	fmt.Println("Type 'exit' to quit")

	chatService := service.NewChatService(
		llmClient,
		executor,
		rt,
	)
	conv := conversation.NewWithSystemPrompt(prompts.DatabaseAgentInstructions)
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

		stream := chatService.Stream(
			context.Background(),
			conv,
			input,
		)

		for result := range stream {

			if result.Err != nil {
				fmt.Println(result.Err)
				break
			}

			fmt.Print(
				result.Chunk.Message.Content,
			)
		}

		fmt.Println()
	}
}
