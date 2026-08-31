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

func TestToolCircuitBreaker_AllowsCallsWhenClosed(t *testing.T) {
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

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 3,
			OpenTimeout:      time.Second,
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
		t.Fatalf("expected 1 call, got %d", got)
	}
}

func TestToolCircuitBreaker_OpensAfterFailureThreshold(t *testing.T) {
	var calls atomic.Int32

	testErr := errors.New("database unavailable")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return nil, testErr
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 3,
			OpenTimeout:      time.Second,
		},
	)(handler)

	for i := 0; i < 3; i++ {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)

		if !errors.Is(err, testErr) {
			t.Fatalf(
				"attempt %d: expected %v, got %v",
				i+1,
				testErr,
				err,
			)
		}
	}

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, ErrToolCircuitOpen) {
		t.Fatalf(
			"expected ErrToolCircuitOpen, got %v",
			err,
		)
	}

	if got := calls.Load(); got != 3 {
		t.Fatalf(
			"expected underlying tool to be called 3 times, got %d",
			got,
		)
	}
}

func TestToolCircuitBreaker_FailsFastWhenOpen(t *testing.T) {
	var calls atomic.Int32

	testErr := errors.New("tool unavailable")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return nil, testErr
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      time.Minute,
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf("expected original error, got %v", err)
	}

	_, err = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, ErrToolCircuitOpen) {
		t.Fatalf(
			"expected ErrToolCircuitOpen, got %v",
			err,
		)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf(
			"expected 1 underlying call, got %d",
			got,
		)
	}
}

func TestToolCircuitBreaker_RecoversAfterOpenTimeout(t *testing.T) {
	var calls atomic.Int32

	testErr := errors.New("temporary failure")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			call := calls.Add(1)

			if call == 1 {
				return nil, testErr
			}

			return &llm.ToolResult{
				Content: "recovered",
			}, nil
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      50 * time.Millisecond,
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf("expected original error, got %v", err)
	}

	_, err = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, ErrToolCircuitOpen) {
		t.Fatalf(
			"expected circuit to be open, got %v",
			err,
		)
	}

	time.Sleep(75 * time.Millisecond)

	result, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("expected recovery, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result")
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf(
			"expected 2 underlying calls, got %d",
			got,
		)
	}
}

func TestToolCircuitBreaker_ReopensAfterFailedProbe(t *testing.T) {
	var calls atomic.Int32

	testErr := errors.New("still unavailable")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return nil, testErr
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      25 * time.Millisecond,
		},
	)(handler)

	_, _ = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	time.Sleep(40 * time.Millisecond)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf(
			"expected probe error, got %v",
			err,
		)
	}

	_, err = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, ErrToolCircuitOpen) {
		t.Fatalf(
			"expected circuit to reopen, got %v",
			err,
		)
	}
}

func TestToolCircuitBreaker_AllowsSingleHalfOpenProbe(t *testing.T) {
	var calls atomic.Int32

	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})

	testErr := errors.New("failure")

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			call := calls.Add(1)

			if call == 1 {
				return nil, testErr
			}

			close(probeStarted)

			<-releaseProbe

			return &llm.ToolResult{
				Content: "recovered",
			}, nil
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      25 * time.Millisecond,
		},
	)(handler)

	_, _ = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	time.Sleep(40 * time.Millisecond)

	const concurrentCalls = 10

	results := make(chan error, concurrentCalls)

	for i := 0; i < concurrentCalls; i++ {
		go func() {
			_, err := wrapped(
				context.Background(),
				ToolInvocation{Name: "test"},
			)

			results <- err
		}()
	}

	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("half-open probe did not start")
	}

	time.Sleep(25 * time.Millisecond)

	if got := calls.Load(); got != 2 {
		t.Fatalf(
			"expected exactly 1 half-open probe, got %d total calls",
			got,
		)
	}

	close(releaseProbe)

	var successCount int

	for i := 0; i < concurrentCalls; i++ {
		select {
		case err := <-results:
			if err == nil {
				successCount++
			} else if !errors.Is(err, ErrToolCircuitOpen) {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

		case <-time.After(time.Second):
			t.Fatal("call did not finish")
		}
	}

	if successCount != 1 {
		t.Fatalf(
			"expected exactly 1 successful probe, got %d",
			successCount,
		)
	}
}

func TestToolCircuitBreaker_DoesNotTripOnCancellation(t *testing.T) {
	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return nil, ctx.Err()
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      time.Minute,
		},
	)(handler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wrapped(
		ctx,
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}

	// A second call with a healthy context must still reach
	// the underlying handler.
	healthyHandler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			return &llm.ToolResult{
				Content: "ok",
			}, nil
		},
	)

	healthyWrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 1,
			OpenTimeout:      time.Minute,
		},
	)(healthyHandler)

	_, err = healthyWrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolCircuitBreaker_IsConcurrencySafe(t *testing.T) {
	testErr := errors.New("failure")

	var calls atomic.Int32

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			calls.Add(1)

			return nil, testErr
		},
	)

	wrapped := ToolCircuitBreaker(
		CircuitBreakerPolicy{
			FailureThreshold: 5,
			OpenTimeout:      time.Second,
		},
	)(handler)

	const concurrentCalls = 100

	var wg sync.WaitGroup

	for i := 0; i < concurrentCalls; i++ {
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

	if got := calls.Load(); got < 5 {
		t.Fatalf(
			"expected at least 5 underlying calls, got %d",
			got,
		)
	}

	if got := calls.Load(); got > concurrentCalls {
		t.Fatalf(
			"invalid call count: %d",
			got,
		)
	}
}
