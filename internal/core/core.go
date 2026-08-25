package core

import "go-llm/internal/events"

type Core struct {
	observer events.Observer
}

func New(
	observer events.Observer,
) *Core {

	if observer == nil {
		observer = events.NopObserver{}
	}

	return &Core{
		observer: observer,
	}
}

func (c *Core) Emit(
	event events.Event,
) {
	c.observer.OnEvent(event)
}
