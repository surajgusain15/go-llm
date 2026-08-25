package events

import "time"

// LLMRequestStarted Defines events emitted before and after an LLM request.
type LLMRequestStarted struct {
	BaseEvent
}

func NewLLMRequestStarted(
	model string,
) LLMRequestStarted {

	return LLMRequestStarted{
		BaseEvent: NewBaseEvent(),
	}
}

func (LLMRequestStarted) Type() EventType {
	return EventLLMRequestStarted
}

type LLMRequestFinished struct {
	BaseEvent

	Duration time.Duration

	Err error
}

func NewLLMRequestFinished(
	model string,
	duration time.Duration,
	err error,
) LLMRequestFinished {

	return LLMRequestFinished{
		BaseEvent: NewBaseEvent(),
		Duration:  duration,
		Err:       err,
	}
}

func (LLMRequestFinished) Type() EventType {
	return EventLLMRequestFinished
}
