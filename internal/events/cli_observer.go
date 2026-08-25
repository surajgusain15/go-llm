package events

import (
	"fmt"
)

type CLIObserver struct{}

func NewCLIObserver() *CLIObserver {
	return &CLIObserver{}
}

func (c *CLIObserver) OnEvent(
	event Event,
) {

	switch e := event.(type) {

	case ToolStarted:

		fmt.Printf(
			"🔧 Tool started: %s\n",
			e.Name,
		)

	case ToolFinished:

		if e.Err != nil {

			fmt.Printf(
				"❌ Tool failed: %s (%v)\n",
				e.Name,
				e.Err,
			)

			return
		}

		fmt.Printf(
			"✅ Tool finished: %s (%s)\n",
			e.Name,
			e.Duration,
		)

	case LLMRequestStarted:

		fmt.Println(
			"🧠 LLM request started...",
		)

	case LLMRequestFinished:

		if e.Err != nil {

			fmt.Printf(
				"❌ LLM request failed: %v\n",
				e.Err,
			)

			return
		}

		fmt.Printf(
			"✅ LLM request completed (%s)\n",
			e.Duration,
		)

	case UserMessage:

		fmt.Printf(
			"\n👤 %s\n",
			e.Content,
		)

	case AssistantMessage:

		fmt.Printf(
			"\n🤖 %s\n",
			e.Content,
		)
	}
}
