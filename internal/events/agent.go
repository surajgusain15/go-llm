package events

import "time"

type AgentIterationStarted struct {
	timestamp time.Time
	Iteration int
}

func NewAgentIterationStarted(
	iteration int,
) AgentIterationStarted {
	return AgentIterationStarted{
		timestamp: time.Now(),
		Iteration: iteration,
	}
}

func (e AgentIterationStarted) Type() EventType {
	return EventAgentIterationStarted
}

func (e AgentIterationStarted) Timestamp() time.Time {
	return e.timestamp
}

func (e AgentIterationStarted) Data() any {
	return e
}

type AgentToolCallsPlanned struct {
	timestamp time.Time
	Calls     []string
}

func NewAgentToolCallsPlanned(
	calls []string,
) AgentToolCallsPlanned {
	return AgentToolCallsPlanned{
		timestamp: time.Now(),
		Calls:     calls,
	}
}

func (e AgentToolCallsPlanned) Type() EventType {
	return EventAgentToolsPlanned
}

func (e AgentToolCallsPlanned) Timestamp() time.Time {
	return e.timestamp
}

func (e AgentToolCallsPlanned) Data() any {
	return e
}

type AgentStarted struct {
	timestamp time.Time
}

func NewAgentStarted() AgentStarted {
	return AgentStarted{
		timestamp: time.Now(),
	}
}

func (e AgentStarted) Type() EventType {
	return EventAgentStarted
}

func (e AgentStarted) Timestamp() time.Time {
	return e.timestamp
}

func (e AgentStarted) Data() any {
	return nil
}

type AgentFinished struct {
	timestamp time.Time
}

func NewAgentFinished() AgentFinished {
	return AgentFinished{
		timestamp: time.Now(),
	}
}

func (e AgentFinished) Type() EventType {
	return EventAgentFinished
}

func (e AgentFinished) Timestamp() time.Time {
	return e.timestamp
}

func (e AgentFinished) Data() any {
	return nil
}

type ThinkingStarted struct {
	timestamp time.Time
}

func NewThinkingStarted() ThinkingStarted {
	return ThinkingStarted{
		timestamp: time.Now(),
	}
}

func (e ThinkingStarted) Type() EventType {
	return EventThinkingStarted
}

func (e ThinkingStarted) Timestamp() time.Time {
	return e.timestamp
}

func (e ThinkingStarted) Data() any {
	return nil
}
