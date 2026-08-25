package events

import "time"

type BaseEvent struct {
	timestamp time.Time
}

func NewBaseEvent() BaseEvent {
	return BaseEvent{
		timestamp: time.Now(),
	}
}

func (b BaseEvent) Timestamp() time.Time {
	return b.timestamp
}
