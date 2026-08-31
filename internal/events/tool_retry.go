package events

import "time"

const EventToolRetry EventType = "tool.retry"

type ToolRetry struct {
	BaseEvent

	Name string

	Attempt int

	Delay time.Duration

	Err error
}

func NewToolRetry(
	name string,
	attempt int,
	delay time.Duration,
	err error,
) ToolRetry {

	return ToolRetry{
		BaseEvent: NewBaseEvent(),
		Name:      name,
		Attempt:   attempt,
		Delay:     delay,
		Err:       err,
	}
}

func (ToolRetry) Type() EventType {
	return EventToolRetry
}
