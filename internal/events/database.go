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

	// True when the query was cancelled because its context
	// deadline expired.
	TimedOut bool

	// True when the query context was cancelled before completion,
	// but not because of a deadline.
	Cancelled bool

	At time.Time
}

func NewDatabaseQueryFinished(
	fingerprint string,
	duration time.Duration,
	rows int,
	err error,
	timedOut bool,
	cancelled bool,
) DatabaseQueryFinished {
	return DatabaseQueryFinished{
		Fingerprint: fingerprint,
		Duration:    duration,
		Rows:        rows,
		Err:         err,
		TimedOut:    timedOut,
		Cancelled:   cancelled,
		At:          time.Now(),
	}
}

func (e DatabaseQueryFinished) Type() EventType {
	return EventDatabaseQueryFinished
}

func (e DatabaseQueryFinished) Timestamp() time.Time {
	return e.At
}
