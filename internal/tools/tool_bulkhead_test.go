package tools

import (
	"context"
	"errors"
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

func TestToolBulkhead_CancelWhileWaiting(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(started)

			<-release

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	wrapped := ToolBulkhead(
		ToolBulkheadPolicy{
			PerTool: map[string]int{
				"test": 1,
			},
		},
	)(handler)

	firstDone := make(chan struct{})

	go func() {
		defer close(firstDone)

		_, _ = wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)
	}()

	select {
	case <-started:

	case <-time.After(time.Second):
		t.Fatal("first execution did not start")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Millisecond,
	)
	defer cancel()

	start := time.Now()

	_, err := wrapped(
		ctx,
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf(
			"expected context deadline exceeded, got %v",
			err,
		)
	}

	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf(
			"bulkhead ignored cancellation: %v",
			elapsed,
		)
	}

	close(release)

	select {
	case <-firstDone:

	case <-time.After(time.Second):
		t.Fatal("first execution did not finish")
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
