package events

import "time"

type DatabaseQueryStarted struct {
	Fingerprint string
	At          time.Time
}

func NewDatabaseQueryStarted(
	fingerprint string,
) DatabaseQueryStarted {
	return DatabaseQueryStarted{
		Fingerprint: fingerprint,
		At:          time.Now(),
	}
}

func (e DatabaseQueryStarted) Type() EventType {
	return EventDatabaseQueryStarted
}

func (e DatabaseQueryStarted) Timestamp() time.Time {
	return e.At
}

type DatabaseQueryFinished struct {
	Fingerprint string
	Duration    time.Duration
	Rows        int
	Err         error
	At          time.Time
}

func NewDatabaseQueryFinished(
	fingerprint string,
	duration time.Duration,
	rows int,
	err error,
) DatabaseQueryFinished {
	return DatabaseQueryFinished{
		Fingerprint: fingerprint,
		Duration:    duration,
		Rows:        rows,
		Err:         err,
		At:          time.Now(),
	}
}

func (e DatabaseQueryFinished) Type() EventType {
	return EventDatabaseQueryFinished
}

func (e DatabaseQueryFinished) Timestamp() time.Time {
	return e.At
}
