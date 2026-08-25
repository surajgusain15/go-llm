package core

import "go-llm/internal/events"

type Core struct {
	Observer events.Observer
}

func New() *Core {
	return &Core{
		Observer: events.NopObserver{},
	}
}
