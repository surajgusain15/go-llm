package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-llm/internal/config"
	"go-llm/internal/core"
	"go-llm/internal/database"
	"go-llm/internal/events"
	"go-llm/internal/tools"
)

func main() {

	cfg := config.Load()
	rt := core.New(events.NewCLIObserver(events.LogLevelDebug))

	db, err := database.NewMySQLClient(
		cfg.MySQLDSN,
		cfg.MySQLQueryTimeout,
		cfg.MySQLMaxRows,
		cfg.MySQLMaxResultBytes,
		cfg.MySQLSchemaCacheTTL,
		rt,
	)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		panic(err)
	}

	executor := tools.NewDefaultExecutor(rt, db)

	runCalculator(executor)

	runCurrentTime(executor)

	runUUID(executor)
}

func runCalculator(
	executor *tools.Executor,
) {

	fmt.Println("=== Calculator ===")

	result, err := executor.Execute(
		context.Background(),
		"calculator",
		json.RawMessage(
			`{
			"a": 10,
			"b": 20
		}`,
		),
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Result:", result)
	fmt.Println()
}

func runCurrentTime(
	executor *tools.Executor,
) {

	fmt.Println("=== Current Time ===")

	result, err := executor.Execute(
		context.Background(),
		"current_time",
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Result:", result)
	fmt.Println()
}

func runUUID(
	executor *tools.Executor,
) {

	fmt.Println("=== UUID ===")

	result, err := executor.Execute(
		context.Background(),
		"uuid",
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Result:", result)
	fmt.Println()
}
