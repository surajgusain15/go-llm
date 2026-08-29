package events

import (
	"fmt"
)

type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelNormal
	LogLevelDebug
	LogLevelInfo
)

type CLIObserver struct {
	LogLevel LogLevel
}

func NewCLIObserver(
	logLevel LogLevel,
) *CLIObserver {
	return &CLIObserver{
		LogLevel: logLevel,
	}
}

func (c *CLIObserver) OnEvent(
	event Event,
) {

	switch e := event.(type) {

	case UserMessage:

		if c.LogLevel < LogLevelInfo {
			return
		}

		fmt.Printf(
			"👤 %s\n",
			e.Content,
		)

	case ThinkingStarted:

		if c.LogLevel < LogLevelInfo {
			return
		}

		fmt.Print(
			"🤖 Thinking...",
		)

	case AgentIterationStarted:

		if c.LogLevel < LogLevelDebug {
			return
		}

		fmt.Printf(
			"\n🤖 Agent iteration %d\n",
			e.Iteration,
		)

	case LLMRequestStarted:

		if c.LogLevel < LogLevelDebug {
			return
		}

		fmt.Print(
			"🧠 LLM request started...\n",
		)

	case ToolStarted:

		if c.LogLevel < LogLevelInfo {
			return
		}

		fmt.Printf(
			"🔧 Using %s...\n",
			e.Name,
		)

	case ToolFinished:

		if c.LogLevel < LogLevelInfo {
			return
		}

		if e.Err != nil {

			fmt.Printf(
				"❌ %s failed (%v)\n",
				e.Name,
				e.Err,
			)

			return
		}

		fmt.Printf(
			"✅ %s completed (%s)\n",
			e.Name,
			e.Duration,
		)

	case LLMRequestFinished:

		if c.LogLevel < LogLevelDebug {
			return
		}

		if e.Err != nil {

			fmt.Printf(
				"❌ LLM request failed: %v\n",
				e.Err,
			)

			return
		}

		fmt.Printf(
			"\n✅ LLM request completed (%s)\n",
			e.Duration,
		)
	}
}
