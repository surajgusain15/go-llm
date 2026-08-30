package database

import (
	"sync"

	"go-llm/internal/events"
)

type recordingObserver struct {
	mu     sync.Mutex
	events []events.Event
}

func (o *recordingObserver) OnEvent(
	event events.Event,
) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.events = append(
		o.events,
		event,
	)
}

func (o *recordingObserver) Events() []events.Event {
	o.mu.Lock()
	defer o.mu.Unlock()

	result := make(
		[]events.Event,
		len(o.events),
	)

	copy(result, o.events)

	return result
}
