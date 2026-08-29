package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go-llm/internal/config"
	"go-llm/internal/core"
	"go-llm/internal/database"
	"go-llm/internal/events"
	"go-llm/internal/llm"
	"go-llm/internal/service"
	"go-llm/internal/tools"
)

func main() {

	cfg := config.Load()

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}

	client := llm.NewOllamaClient(
		httpClient,
		cfg.OllamaBaseURL,
		cfg.Model,
	)

	db, err := database.NewMySQLClient(
		cfg.MySQLDSN,
		cfg.MySQLQueryTimeout,
		cfg.MySQLMaxRows,
	)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		panic(err)
	}

	rt := core.New(events.NewCLIObserver(events.LogLevelDebug))

	executor := tools.NewDefaultExecutor(rt, db)

	chatService := service.NewChatService(client, executor, rt)

	conv := chatService.NewConversation()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== Tool Calling Demo ===")
	fmt.Println("Type 'exit' to quit.")
	fmt.Println()

	for {

		fmt.Print("You: ")

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

		reply, err := chatService.Chat(
			context.Background(),
			conv,
			input,
		)
		if err != nil {
			log.Println(err)
			continue
		}

		fmt.Println("Assistant:", reply)
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
