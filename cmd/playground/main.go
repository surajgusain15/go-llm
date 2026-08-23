package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go-llm/internal/tools"
)

func main() {

	executor := tools.NewDefaultExecutor()

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
			err != tools.ErrToolNotInteractive {

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
