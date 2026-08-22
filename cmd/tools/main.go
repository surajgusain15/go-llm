package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go-llm/internal/tools"
	"go-llm/internal/tools/builtin"
)

func main() {

	registry := tools.NewRegistry()

	registry.Register(
		builtin.NewTimeTool(),
	)

	registry.Register(
		builtin.NewUUIDTool(),
	)

	registry.Register(
		builtin.NewCalculatorTool(),
	)

	executor := tools.NewExecutor(
		registry,
	)

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
