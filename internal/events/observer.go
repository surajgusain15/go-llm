package events

type Observer interface {
	OnEvent(Event)
}
