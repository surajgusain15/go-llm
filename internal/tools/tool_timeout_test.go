package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-llm/internal/llm"
)

func TestToolTimeout_ParentCancellation(
	t *testing.T,
) {
	executed := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(executed)

			<-ctx.Done()

			return nil, ctx.Err()
		},
	)

	middleware := ToolTimeout(
		5 * time.Second,
	)

	wrapped := middleware(handler)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	done := make(chan error, 1)

	go func() {
		_, err := wrapped(
			ctx,
			ToolInvocation{
				Name: "test",
			},
		)

		done <- err
	}()

	select {
	case <-executed:
	case <-time.After(time.Second):
		t.Fatal("tool did not start")
	}

	cancel()

	select {
	case err := <-done:

		if !errors.Is(
			err,
			context.Canceled,
		) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"tool did not terminate after parent cancellation",
		)
	}
}

func TestToolTimeout_Expires(
	t *testing.T,
) {
	executed := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(executed)

			<-ctx.Done()

			return nil, ctx.Err()
		},
	)

	timeout := 50 * time.Millisecond

	wrapped := ToolTimeout(timeout)(
		handler,
	)

	start := time.Now()

	_, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf(
			"expected context.DeadlineExceeded, got %v",
			err,
		)
	}

	select {
	case <-executed:
	default:
		t.Fatal("tool did not execute")
	}

	// Make sure the timeout actually happened rather than
	// the handler returning immediately.
	if elapsed < timeout {
		t.Fatalf(
			"tool returned too early: elapsed %v, timeout %v",
			elapsed,
			timeout,
		)
	}
}

func TestToolTimeout_ConcurrentToolsHaveIndependentTimeouts(
	t *testing.T,
) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	firstFinished := make(chan error, 1)
	secondFinished := make(chan error, 1)

	firstHandler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(firstStarted)

			<-ctx.Done()

			firstFinished <- ctx.Err()

			return nil, ctx.Err()
		},
	)

	secondHandler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			close(secondStarted)

			<-ctx.Done()

			secondFinished <- ctx.Err()

			return nil, ctx.Err()
		},
	)

	firstWrapped := ToolTimeout(
		50 * time.Millisecond,
	)(firstHandler)

	secondWrapped := ToolTimeout(
		200 * time.Millisecond,
	)(secondHandler)

	go func() {
		_, _ = firstWrapped(
			context.Background(),
			ToolInvocation{Name: "first"},
		)
	}()

	go func() {
		_, _ = secondWrapped(
			context.Background(),
			ToolInvocation{Name: "second"},
		)
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first tool did not start")
	}

	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("second tool did not start")
	}

	// First tool must timeout independently.
	select {
	case err := <-firstFinished:
		if !errors.Is(
			err,
			context.DeadlineExceeded,
		) {
			t.Fatalf(
				"first tool: expected context.DeadlineExceeded, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first tool did not timeout")
	}

	// The second tool must still be running after
	// the first tool's timeout.
	select {
	case err := <-secondFinished:
		t.Fatalf(
			"second tool finished when first timed out: %v",
			err,
		)

	case <-time.After(75 * time.Millisecond):
		// Expected: second tool is still alive.
	}

	// Eventually the second tool gets its own timeout.
	select {
	case err := <-secondFinished:
		if !errors.Is(
			err,
			context.DeadlineExceeded,
		) {
			t.Fatalf(
				"second tool: expected context.DeadlineExceeded, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second tool did not timeout")
	}
}
