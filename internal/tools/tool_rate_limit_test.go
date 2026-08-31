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

func TestToolRateLimit_AllowsBurst(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 3,
			Rate:  10 * time.Second,
		},
	)(handler)

	for i := 0; i < 3; i++ {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)

		if err != nil {
			t.Fatalf(
				"call %d: unexpected error: %v",
				i+1,
				err,
			)
		}
	}

	if got := calls.Load(); got != 3 {
		t.Fatalf(
			"expected 3 calls, got %d",
			got,
		)
	}
}

func TestToolRateLimit_ThrottlesAfterBurst(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 1,
			Rate:  100 * time.Millisecond,
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	start := time.Now()

	_, err = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed < 70*time.Millisecond {
		t.Fatalf(
			"expected rate limiting delay, got %v",
			elapsed,
		)
	}
}

func TestToolRateLimit_ReplenishesTokens(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 1,
			Rate:  50 * time.Millisecond,
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	start := time.Now()

	_, err = wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 30*time.Millisecond {
		t.Fatalf(
			"expected replenished token, waited %v",
			elapsed,
		)
	}
}

func TestToolRateLimit_CancelWhileWaiting(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 1,
			Rate:  time.Second,
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{Name: "test"},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Millisecond,
	)
	defer cancel()

	start := time.Now()

	_, err = wrapped(
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
			"rate limiter ignored cancellation: %v",
			elapsed,
		)
	}
}

func TestToolRateLimit_DisabledForInvalidPolicy(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 0,
			Rate:  time.Second,
		},
	)(handler)

	for i := 0; i < 3; i++ {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{Name: "test"},
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if got := calls.Load(); got != 3 {
		t.Fatalf(
			"expected 3 calls, got %d",
			got,
		)
	}
}

func TestToolRateLimit_IsConcurrencySafe(t *testing.T) {
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

	wrapped := ToolRateLimit(
		RateLimitPolicy{
			Burst: 100,
			Rate:  1000 * time.Second,
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

	if got := calls.Load(); got != concurrentCalls {
		t.Fatalf(
			"expected %d calls, got %d",
			concurrentCalls,
			got,
		)
	}
}
