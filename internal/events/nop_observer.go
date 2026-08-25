package events

type NopObserver struct{}

func (NopObserver) OnEvent(Event) {}
