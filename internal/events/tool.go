package events

import "time"

type ToolStarted struct {
	BaseEvent

	Name string
}

func NewToolStarted(
	name string,
) ToolStarted {

	return ToolStarted{
		BaseEvent: NewBaseEvent(),
		Name:      name,
	}
}

func (ToolStarted) Type() EventType {
	return EventToolStarted
}

type ToolFinished struct {
	BaseEvent

	Name string

	Duration time.Duration

	Err error
}

func NewToolFinished(
	name string,
	duration time.Duration,
	err error,
) ToolFinished {

	return ToolFinished{
		BaseEvent: NewBaseEvent(),
		Name:      name,
		Duration:  duration,
		Err:       err,
	}
}

func (ToolFinished) Type() EventType {
	return EventToolFinished
}
