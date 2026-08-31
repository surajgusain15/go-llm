package tools

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-llm/internal/llm"
)

func TestToolBulkhead_LimitsPerToolConcurrency(t *testing.T) {
	var current atomic.Int32
	var maximum atomic.Int32

	started := make(chan struct{}, 10)
	release := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			value := current.Add(1)

			for {
				old := maximum.Load()

				if value <= old {
					break
				}

				if maximum.CompareAndSwap(old, value) {
					break
				}
			}

			started <- struct{}{}

			<-release

			current.Add(-1)

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"database_query": 2,
			},
		},
	)(handler)

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = wrapped(
				context.Background(),
				ToolInvocation{
					Name: "database_query",
				},
			)
		}()
	}

	// Exactly two should enter the underlying handler.
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected tool execution to start")
		}
	}

	select {
	case <-started:
		t.Fatal("third execution exceeded per-tool limit")

	case <-time.After(50 * time.Millisecond):
	}

	if got := maximum.Load(); got != 2 {
		t.Fatalf(
			"expected maximum concurrency 2, got %d",
			got,
		)
	}

	close(release)

	wg.Wait()
}

func TestToolBulkhead_IsolatesTools(t *testing.T) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	releaseFirst := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			switch invocation.Name {
			case "database_query":
				close(firstStarted)

				<-releaseFirst

			case "payment":
				close(secondStarted)
			}

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"database_query": 1,
				"payment":        1,
			},
		},
	)(handler)

	firstDone := make(chan struct{})

	go func() {
		defer close(firstDone)

		_, _ = wrapped(
			context.Background(),
			ToolInvocation{
				Name: "database_query",
			},
		)
	}()

	select {
	case <-firstStarted:

	case <-time.After(time.Second):
		t.Fatal("first tool did not start")
	}

	secondDone := make(chan struct{})

	go func() {
		defer close(secondDone)

		_, _ = wrapped(
			context.Background(),
			ToolInvocation{
				Name: "payment",
			},
		)
	}()

	select {
	case <-secondStarted:

	case <-time.After(time.Second):
		t.Fatal("second tool was starved by first tool")
	}

	close(releaseFirst)

	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first tool did not finish")
	}

	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second tool did not finish")
	}
}

func TestToolBulkhead_CancelWhileWaiting(
	t *testing.T,
) {
	var executions atomic.Int32

	release := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			executions.Add(1)

			select {
			case <-release:
				return &llm.ToolResult{}, nil

			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"test": 1,
			},
		},
	)(handler)

	firstDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)

		firstDone <- err
	}()

	// Wait until first execution has acquired the slot.
	deadline := time.After(time.Second)

	for executions.Load() != 1 {
		select {
		case <-deadline:
			t.Fatal("first tool did not start")
		default:
			runtime.Gosched()
		}
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	secondDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			ctx,
			ToolInvocation{Name: "test"},
		)

		secondDone <- err
	}()

	// Let the second invocation reach the bulkhead.
	time.Sleep(20 * time.Millisecond)

	cancel()

	select {
	case err := <-secondDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"waiting invocation did not terminate after cancellation",
		)
	}

	if got := executions.Load(); got != 1 {
		t.Fatalf(
			"expected exactly 1 tool execution, got %d",
			got,
		)
	}

	close(release)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf(
				"first tool: unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first tool did not finish")
	}
}

func TestToolBulkhead_UnconfiguredToolPassesThrough(t *testing.T) {
	var calls atomic.Int32

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"database_query": 1,
			},
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "payment",
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf(
			"expected 1 call, got %d",
			got,
		)
	}
}

func TestToolBulkhead_InvalidLimitPassesThrough(t *testing.T) {
	var calls atomic.Int32

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"test": 0,
			},
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf(
			"expected 1 call, got %d",
			got,
		)
	}
}

func TestToolBulkhead_IsConcurrencySafe(t *testing.T) {
	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"test": 10,
			},
		},
	)(handler)

	const calls = 100

	var wg sync.WaitGroup

	for i := 0; i < calls; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _ = wrapped(
				context.Background(),
				ToolInvocation{Name: "test"},
			)
		}()
	}

	wg.Wait()
}

func TestToolBulkhead_DifferentToolsHaveIndependentLimits(
	t *testing.T,
) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	releaseFirst := make(chan struct{})

	firstHandler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(firstStarted)

			select {
			case <-releaseFirst:
				return &llm.ToolResult{}, nil

			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	)

	secondHandler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(secondStarted)

			return &llm.ToolResult{}, nil
		},
	)

	middleware := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"first":  1,
				"second": 1,
			},
		},
	)

	firstWrapped := middleware(firstHandler)
	secondWrapped := middleware(secondHandler)

	firstDone := make(chan error, 1)

	go func() {
		_, err := firstWrapped(
			context.Background(),
			ToolInvocation{
				Name: "first",
			},
		)

		firstDone <- err
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first tool did not start")
	}

	secondDone := make(chan error, 1)

	go func() {
		_, err := secondWrapped(
			context.Background(),
			ToolInvocation{
				Name: "second",
			},
		)

		secondDone <- err
	}()

	select {
	case <-secondStarted:
		// Correct: second has its own bulkhead.
	case <-time.After(time.Second):
		t.Fatal(
			"second tool was blocked by first tool's bulkhead",
		)
	}

	close(releaseFirst)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf(
				"first tool: unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first tool did not finish")
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf(
				"second tool: unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second tool did not finish")
	}
}

func TestToolBulkhead_SlotReleasedAfterError(
	t *testing.T,
) {
	var executions atomic.Int32

	testErr := errors.New("tool failure")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			executions.Add(1)

			return nil, testErr
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"test": 1,
			},
		},
	)(handler)

	// First execution fails while holding the only slot.
	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf(
			"expected original tool error, got %v",
			err,
		)
	}

	// The slot must have been released even though
	// the handler returned an error.
	done := make(chan error, 1)

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)

		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"second invocation: expected original tool error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"second invocation blocked; bulkhead slot was leaked",
		)
	}

	if got := executions.Load(); got != 2 {
		t.Fatalf(
			"expected 2 executions, got %d",
			got,
		)
	}
}
