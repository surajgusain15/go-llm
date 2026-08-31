package tools

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

type recordingToolObserver struct {
	mu     sync.Mutex
	events []events.Event
}

func (r *recordingToolObserver) OnEvent(
	event events.Event,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(
		r.events,
		event,
	)
}

func (r *recordingToolObserver) snapshot() []events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make(
		[]events.Event,
		len(r.events),
	)

	copy(result, r.events)

	return result
}

func TestToolObservability_EmitsStartedAndFinished(
	t *testing.T,
) {
	observer := &recordingToolObserver{}
	rt := core.New(observer)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return &llm.ToolResult{}, nil
		},
	)

	wrapped := ToolObservability()(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	_, err := wrapped(
		ctx,
		ToolInvocation{
			Name: "database_query",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected nil error, got %v",
			err,
		)
	}

	recorded := observer.snapshot()

	if len(recorded) != 2 {
		t.Fatalf(
			"expected 2 events, got %d",
			len(recorded),
		)
	}

	if recorded[0].Type() != events.EventToolStarted {
		t.Fatalf(
			"expected first event tool.started, got %s",
			recorded[0].Type(),
		)
	}

	if recorded[1].Type() != events.EventToolFinished {
		t.Fatalf(
			"expected second event tool.finished, got %s",
			recorded[1].Type(),
		)
	}

	started, ok := recorded[0].(events.ToolStarted)

	if !ok {
		t.Fatalf(
			"expected ToolStarted, got %T",
			recorded[0],
		)
	}

	if started.Name != "database_query" {
		t.Fatalf(
			"expected tool name database_query, got %q",
			started.Name,
		)
	}

	finished, ok := recorded[1].(events.ToolFinished)

	if !ok {
		t.Fatalf(
			"expected ToolFinished, got %T",
			recorded[1],
		)
	}

	if finished.Name != "database_query" {
		t.Fatalf(
			"expected tool name database_query, got %q",
			finished.Name,
		)
	}

	if finished.Err != nil {
		t.Fatalf(
			"expected nil finished error, got %v",
			finished.Err,
		)
	}

	if finished.Duration <= 0 {
		t.Fatalf(
			"expected positive duration, got %v",
			finished.Duration,
		)
	}
}

func TestToolObservability_EmitsFinishedOnError(
	t *testing.T,
) {
	expectedErr := errors.New("tool failed")

	observer := &recordingToolObserver{}
	rt := core.New(observer)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return nil, expectedErr
		},
	)

	wrapped := ToolObservability()(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	_, err := wrapped(
		ctx,
		ToolInvocation{
			Name: "database_query",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected original error, got %v",
			err,
		)
	}

	recorded := observer.snapshot()

	if len(recorded) != 2 {
		t.Fatalf(
			"expected 2 events, got %d",
			len(recorded),
		)
	}

	finished, ok := recorded[1].(events.ToolFinished)

	if !ok {
		t.Fatalf(
			"expected ToolFinished, got %T",
			recorded[1],
		)
	}

	if !errors.Is(finished.Err, expectedErr) {
		t.Fatalf(
			"expected finished error %v, got %v",
			expectedErr,
			finished.Err,
		)
	}
}

func TestToolObservability_DoesNotEmitWithoutCore(
	t *testing.T,
) {
	called := false

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			called = true

			return &llm.ToolResult{}, nil
		},
	)

	wrapped := ToolObservability()(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected nil error, got %v",
			err,
		)
	}

	if !called {
		t.Fatal("handler was not called")
	}
}

func TestToolObservability_RecordsActualDuration(
	t *testing.T,
) {
	observer := &recordingToolObserver{}
	rt := core.New(observer)

	const executionTime = 30 * time.Millisecond

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			time.Sleep(executionTime)

			return &llm.ToolResult{}, nil
		},
	)

	wrapped := ToolObservability()(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	_, err := wrapped(
		ctx,
		ToolInvocation{
			Name: "slow_tool",
		},
	)

	if err != nil {
		t.Fatalf(
			"expected nil error, got %v",
			err,
		)
	}

	recorded := observer.snapshot()

	if len(recorded) != 2 {
		t.Fatalf(
			"expected 2 events, got %d",
			len(recorded),
		)
	}

	finished, ok := recorded[1].(events.ToolFinished)

	if !ok {
		t.Fatalf(
			"expected ToolFinished, got %T",
			recorded[1],
		)
	}

	if finished.Duration < executionTime {
		t.Fatalf(
			"expected duration >= %v, got %v",
			executionTime,
			finished.Duration,
		)
	}
}
