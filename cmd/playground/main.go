package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"go-llm/internal/config"
	"go-llm/internal/core"
	"go-llm/internal/database"
	"go-llm/internal/events"
	"go-llm/internal/tools"
)

func main() {

	cfg := config.Load()

	db, err := database.NewMySQLClient(
		cfg.MySQLDSN,
		cfg.MySQLQueryTimeout,
		cfg.MySQLMaxRows,
		cfg.MySQLMaxResultBytes,
		cfg.MySQLSchemaCacheTTL,
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

	reader := bufio.NewReader(os.Stdin)

	for {

		fmt.Println()
		fmt.Println("========== AI Playground ==========")

		for _, tool := range executor.List() {

			fmt.Printf(
				"- %-20s %s\n",
				tool.Name,
				tool.Description,
			)
		}

		fmt.Println("- exit")

		fmt.Print("\nSelect tool: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "exit" {
			return
		}

		payload, err := executor.CollectInput(
			input,
			reader,
		)

		if err != nil &&
			!errors.Is(err, tools.ErrToolNotInteractive) {

			fmt.Println(err)
			continue
		}

		result, err := executor.Execute(
			context.Background(),
			input,
			payload,
		)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("\nResult:\n%v\n\n", result)
	}
}
