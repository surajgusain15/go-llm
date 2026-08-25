package events

import (
	"fmt"
)

type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelNormal
	LogLevelDebug
)

type CLIObserver struct {
	Level LogLevel
}

func NewCLIObserver(level LogLevel) *CLIObserver {
	return &CLIObserver{Level: level}
}

func (c *CLIObserver) enabled(
	level LogLevel,
) bool {

	return c.Level >= level
}

func (c *CLIObserver) OnEvent(
	event Event,
) {

	switch e := event.(type) {

	case AgentStarted:

		if c.enabled(LogLevelNormal) {
			fmt.Println("🤖 Thinking...")
		}

	case AgentIterationStarted:

		if c.enabled(LogLevelDebug) {
			fmt.Printf(
				"🤖 Agent iteration %d\n",
				e.Iteration,
			)
		}

	case ToolStarted:

		if c.enabled(LogLevelNormal) {
			fmt.Printf(
				"🔧 Using %s...\n",
				e.Name,
			)
		}

	case ToolFinished:

		if c.enabled(LogLevelDebug) {

			if e.Err != nil {

				fmt.Printf(
					"❌ %s failed (%v)\n",
					e.Name,
					e.Err,
				)

			} else {

				fmt.Printf(
					"✅ %s completed (%s)\n",
					e.Name,
					e.Duration,
				)
			}
		}

	case LLMRequestStarted:

		if c.enabled(LogLevelDebug) {
			fmt.Println("🧠 LLM request started...")
		}

	case LLMRequestFinished:

		if c.enabled(LogLevelDebug) {

			if e.Err != nil {

				fmt.Printf(
					"❌ LLM request failed: %v\n",
					e.Err,
				)

			} else {

				fmt.Printf(
					"✅ LLM request completed (%s)\n",
					e.Duration,
				)
			}
		}

	case AssistantMessage:
		// Ignore for streaming CLI.
	}
}
