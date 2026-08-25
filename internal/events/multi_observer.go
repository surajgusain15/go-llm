package events

type MultiObserver struct {
	observers []Observer
}

func NewMultiObserver(
	observers ...Observer,
) *MultiObserver {

	return &MultiObserver{
		observers: observers,
	}
}
