package tools

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"
	"go-llm/internal/llm"
)

var errTransientTest = errors.New(
	"transient test failure",
)

func TestToolRetry_TransientErrorThenSuccess(
	t *testing.T,
) {
	var (
		mu       sync.Mutex
		attempts int
	)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			mu.Lock()
			attempts++
			current := attempts
			mu.Unlock()

			if current == 1 {
				return nil, errTransientTest
			}

			return &llm.ToolResult{
				Content: "success",
			}, nil
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 3,

			ShouldRetry: func(err error) bool {
				return errors.Is(
					err,
					errTransientTest,
				)
			},
		},
	)(handler)

	result, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal("expected result")
	}

	if result.Content != "success" {
		t.Fatalf(
			"expected success, got %q",
			result.Content,
		)
	}

	mu.Lock()
	gotAttempts := attempts
	mu.Unlock()

	if gotAttempts != 2 {
		t.Fatalf(
			"expected 2 attempts, got %d",
			gotAttempts,
		)
	}
}

func TestToolRetry_DoesNotRetryNonRetryableError(
	t *testing.T,
) {
	var attempts int

	errPermanent := errors.New(
		"invalid tool input",
	)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			attempts++

			return nil, errPermanent
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 3,

			ShouldRetry: func(err error) bool {
				return errors.Is(
					err,
					errTransientTest,
				)
			},
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if !errors.Is(err, errPermanent) {
		t.Fatalf(
			"expected permanent error, got %v",
			err,
		)
	}

	if attempts != 1 {
		t.Fatalf(
			"expected exactly 1 attempt, got %d",
			attempts,
		)
	}
}

func TestToolRetry_DoesNotRetryCancellation(
	t *testing.T,
) {
	var attempts int

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			attempts++

			return nil, ctx.Err()
		},
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 5,

			ShouldRetry: func(err error) bool {
				return true
			},
		},
	)(handler)

	_, err := wrapped(
		ctx,
		ToolInvocation{
			Name: "test",
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}

	if attempts != 1 {
		t.Fatalf(
			"expected exactly 1 attempt, got %d",
			attempts,
		)
	}
}

func TestExponentialBackoff(t *testing.T) {
	backoff := ExponentialBackoff(
		100*time.Millisecond,
		2*time.Second,
	)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1600 * time.Millisecond},
		{6, 2 * time.Second},
		{7, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(
			fmt.Sprintf("attempt_%d", tt.attempt),
			func(t *testing.T) {

				got := backoff(tt.attempt)

				if got != tt.want {
					t.Fatalf(
						"expected %v, got %v",
						tt.want,
						got,
					)
				}
			},
		)
	}
}

func TestExponentialBackoff_InvalidAttempt(
	t *testing.T,
) {
	backoff := ExponentialBackoff(
		100*time.Millisecond,
		2*time.Second,
	)

	for _, attempt := range []int{-1, 0} {
		if got := backoff(attempt); got != 0 {
			t.Fatalf(
				"attempt %d: expected 0, got %v",
				attempt,
				got,
			)
		}
	}
}

func TestExponentialBackoff_InvalidBase(
	t *testing.T,
) {
	backoff := ExponentialBackoff(
		0,
		2*time.Second,
	)

	if got := backoff(1); got != 0 {
		t.Fatalf(
			"expected 0, got %v",
			got,
		)
	}
}

func TestToolRetry_UsesBackoffPolicy(
	t *testing.T,
) {
	var (
		mu       sync.Mutex
		attempts []int
	)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			mu.Lock()
			attempts = append(
				attempts,
				len(attempts)+1,
			)
			mu.Unlock()

			return nil, errTransientTest
		},
	)

	var backoffAttempts []int

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 3,

			ShouldRetry: func(err error) bool {
				return errors.Is(
					err,
					errTransientTest,
				)
			},

			Backoff: func(attempt int) time.Duration {
				backoffAttempts = append(
					backoffAttempts,
					attempt,
				)

				return 0
			},
		},
	)(handler)

	_, err := wrapped(
		context.Background(),
		ToolInvocation{
			Name: "test",
		},
	)

	if !errors.Is(err, errTransientTest) {
		t.Fatalf(
			"expected transient error, got %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		backoffAttempts,
		[]int{1, 2},
	) {
		t.Fatalf(
			"expected backoff attempts [1 2], got %v",
			backoffAttempts,
		)
	}

	if len(attempts) != 3 {
		t.Fatalf(
			"expected 3 attempts, got %d",
			len(attempts),
		)
	}
}

func TestToolRetry_DoesNotHoldConcurrencySlotDuringBackoff(t *testing.T) {
	releaseFirst := make(chan struct{})
	firstAttempt := make(chan struct{})
	secondToolStarted := make(chan struct{})

	var firstOnce sync.Once

	var (
		mu       sync.Mutex
		attempts int
	)

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			if invocation.Name == "first" {
				mu.Lock()
				attempts++
				attempt := attempts
				mu.Unlock()

				if attempt == 1 {
					firstOnce.Do(
						func() {
							close(firstAttempt)
						},
					)

					<-releaseFirst

					return nil, errTransientTest
				}

				return &llm.ToolResult{
					Content: "first retry succeeded",
				}, nil
			}

			if invocation.Name == "second" {
				close(secondToolStarted)

				return &llm.ToolResult{
					Content: "second",
				}, nil
			}

			return nil, nil
		},
	)

	concurrent := ToolConcurrency(1)(handler)

	retry := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,
			ShouldRetry: func(err error) bool {
				return errors.Is(err, errTransientTest)
			},
			Backoff: func(attempt int) time.Duration {
				return 200 * time.Millisecond
			},
		},
	)(concurrent)

	firstDone := make(chan error, 1)

	go func() {
		_, err := retry(
			context.Background(),
			ToolInvocation{
				Name: "first",
			},
		)

		firstDone <- err
	}()

	select {
	case <-firstAttempt:
	case <-time.After(time.Second):
		t.Fatal("first tool did not start")
	}

	// Make the first attempt return an error. Retry will then
	// enter its backoff period.
	close(releaseFirst)

	// Give the first invocation a chance to enter retry backoff.
	time.Sleep(25 * time.Millisecond)

	secondDone := make(chan error, 1)

	go func() {
		_, err := retry(
			context.Background(),
			ToolInvocation{
				Name: "second",
			},
		)

		secondDone <- err
	}()

	// The second invocation must acquire the concurrency slot
	// while the first invocation is sleeping in retry backoff.
	select {
	case <-secondToolStarted:
	case <-time.After(time.Second):
		t.Fatal(
			"second tool could not acquire slot during retry backoff",
		)
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second tool failed: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("second tool did not finish")
	}

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf(
				"first tool returned unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first tool did not finish")
	}
}

func TestToolRetry_ReleasesConcurrencySlotBeforeBackoff(
	t *testing.T,
) {
	var (
		mu       sync.Mutex
		attempts int
	)

	firstFailed := make(chan struct{})
	secondStarted := make(chan struct{})

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			if invocation.Name == "retrying" {

				mu.Lock()
				attempts++
				attempt := attempts
				mu.Unlock()

				if attempt == 1 {
					close(firstFailed)

					return nil, errTransientTest
				}

				return &llm.ToolResult{
					Content: "retry success",
				}, nil
			}

			close(secondStarted)

			return &llm.ToolResult{
				Content: "second success",
			}, nil
		},
	)

	concurrent := ToolConcurrency(1)(handler)

	retry := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,

			ShouldRetry: func(err error) bool {
				return errors.Is(
					err,
					errTransientTest,
				)
			},

			Backoff: func(attempt int) time.Duration {
				return 200 * time.Millisecond
			},
		},
	)(concurrent)

	retryDone := make(chan error, 1)

	go func() {
		_, err := retry(
			context.Background(),
			ToolInvocation{
				Name: "retrying",
			},
		)

		retryDone <- err
	}()

	select {
	case <-firstFailed:
	case <-time.After(time.Second):
		t.Fatal("first attempt did not execute")
	}

	secondDone := make(chan error, 1)

	go func() {
		_, err := concurrent(
			context.Background(),
			ToolInvocation{
				Name: "second",
			},
		)

		secondDone <- err
	}()

	select {
	case <-secondStarted:
		// Correct: retry is currently sleeping.

	case <-time.After(50 * time.Millisecond):
		t.Fatal(
			"second tool could not acquire slot during retry backoff",
		)
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf(
				"second tool failed: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second tool did not finish")
	}

	select {
	case err := <-retryDone:
		if err != nil {
			t.Fatalf(
				"retrying tool failed: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("retrying tool did not finish")
	}
}

func TestToolRetry_CancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu       sync.Mutex
		attempts int
	)

	firstAttemptStarted := make(chan struct{})
	firstAttemptOnce := sync.Once{}

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {

			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt == 1 {
				firstAttemptOnce.Do(
					func() {
						close(firstAttemptStarted)
					},
				)
			}

			return nil, errTransientTest
		},
	)

	retry := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 3,
			ShouldRetry: func(err error) bool {
				return errors.Is(err, errTransientTest)
			},
			Backoff: func(attempt int) time.Duration {
				return time.Second
			},
		},
	)(handler)

	done := make(chan error, 1)

	go func() {
		_, err := retry(
			ctx,
			ToolInvocation{
				Name: "test",
			},
		)

		done <- err
	}()

	select {
	case <-firstAttemptStarted:
	case <-time.After(time.Second):
		t.Fatal("first attempt did not execute")
	}

	// Cancel while retry is in its backoff period.
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("retry did not stop after cancellation")
	}

	mu.Lock()
	gotAttempts := attempts
	mu.Unlock()

	if gotAttempts != 1 {
		t.Fatalf(
			"expected exactly 1 attempt, got %d",
			gotAttempts,
		)
	}
}

type recordingToolRetryObserver struct {
	mu     sync.Mutex
	events []events.Event
}

func (o *recordingToolRetryObserver) OnEvent(event events.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.events = append(o.events, event)
}

func (o *recordingToolRetryObserver) Events() []events.Event {
	o.mu.Lock()
	defer o.mu.Unlock()

	result := make([]events.Event, len(o.events))
	copy(result, o.events)

	return result
}

func TestToolRetry_EmitsRetryEvent(t *testing.T) {
	testErr := errors.New("temporary failure")

	observer := &recordingToolRetryObserver{}
	rt := core.New(observer)

	attempts := 0

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			attempts++

			if attempts == 1 {
				return nil, testErr
			}

			return &llm.ToolResult{}, nil
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,
			ShouldRetry: func(err error) bool {
				return errors.Is(err, testErr)
			},
			Backoff: func(attempt int) time.Duration {
				return 25 * time.Millisecond
			},
		},
	)(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	result, err := wrapped(
		ctx,
		ToolInvocation{
			Name: "test",
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result")
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}

	var retries []events.ToolRetry

	for _, event := range observer.Events() {
		if retry, ok := event.(events.ToolRetry); ok {
			retries = append(retries, retry)
		}
	}

	if len(retries) != 1 {
		t.Fatalf(
			"expected 1 ToolRetry event, got %d",
			len(retries),
		)
	}

	retry := retries[0]

	if retry.Name != "test" {
		t.Fatalf(
			"expected tool name %q, got %q",
			"test",
			retry.Name,
		)
	}

	if retry.Attempt != 1 {
		t.Fatalf(
			"expected attempt 1, got %d",
			retry.Attempt,
		)
	}

	if retry.Delay != 25*time.Millisecond {
		t.Fatalf(
			"expected delay %v, got %v",
			25*time.Millisecond,
			retry.Delay,
		)
	}

	if !errors.Is(retry.Err, testErr) {
		t.Fatalf(
			"expected error %v, got %v",
			testErr,
			retry.Err,
		)
	}
}

func TestToolRetry_DoesNotEmitRetryEventForNonRetryableError(
	t *testing.T,
) {
	testErr := errors.New("permanent failure")

	observer := &recordingToolRetryObserver{}
	rt := core.New(observer)

	attempts := 0

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			attempts++

			return nil, testErr
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 5,
			ShouldRetry: func(err error) bool {
				return false
			},
			Backoff: func(attempt int) time.Duration {
				return 25 * time.Millisecond
			},
		},
	)(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	_, err := wrapped(
		ctx,
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf(
			"expected original error, got %v",
			err,
		)
	}

	if attempts != 1 {
		t.Fatalf(
			"expected 1 attempt, got %d",
			attempts,
		)
	}

	for _, event := range observer.Events() {
		if _, ok := event.(events.ToolRetry); ok {
			t.Fatal(
				"unexpected ToolRetry event for non-retryable error",
			)
		}
	}
}

func TestToolRetry_DoesNotEmitRetryEventAfterFinalAttempt(
	t *testing.T,
) {
	testErr := errors.New("still failing")

	observer := &recordingToolRetryObserver{}
	rt := core.New(observer)

	attempts := 0

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			attempts++

			return nil, testErr
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,
			ShouldRetry: func(err error) bool {
				return true
			},
			Backoff: func(attempt int) time.Duration {
				return 1 * time.Millisecond
			},
		},
	)(handler)

	ctx := withToolMiddlewareContext(
		context.Background(),
		rt,
	)

	_, err := wrapped(
		ctx,
		ToolInvocation{Name: "test"},
	)

	if !errors.Is(err, testErr) {
		t.Fatalf(
			"expected final tool error, got %v",
			err,
		)
	}

	if attempts != 2 {
		t.Fatalf(
			"expected 2 attempts, got %d",
			attempts,
		)
	}

	var retries []events.ToolRetry

	for _, event := range observer.Events() {
		if retry, ok := event.(events.ToolRetry); ok {
			retries = append(retries, retry)
		}
	}

	if len(retries) != 1 {
		t.Fatalf(
			"expected exactly 1 ToolRetry event, got %d",
			len(retries),
		)
	}

	if retries[0].Attempt != 1 {
		t.Fatalf(
			"expected retry event for attempt 1, got %d",
			retries[0].Attempt,
		)
	}
}

func TestToolRetryBudget_LimitsPerToolRetries(t *testing.T) {
	testErr := errors.New("temporary failure")

	firstBackoffEntered := make(chan struct{})
	releaseFirstBackoff := make(chan struct{})
	secondBackoffEntered := make(chan struct{})

	var mu sync.Mutex
	backoffCalls := 0

	backoff := func(attempt int) time.Duration {
		mu.Lock()
		backoffCalls++
		call := backoffCalls
		mu.Unlock()

		if call == 1 {
			close(firstBackoffEntered)

			select {
			case <-releaseFirstBackoff:
			case <-time.After(time.Second):
				t.Fatal("first backoff was not released")
			}
		}

		if call == 2 {
			close(secondBackoffEntered)
		}

		return 0
	}

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			return nil, testErr
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,

			ShouldRetry: func(err error) bool {
				return errors.Is(err, testErr)
			},

			Backoff: backoff,

			Budget: &ToolRetryBudgetPolicy{
				PerTool: map[string]int{
					"test": 1,
				},
			},
		},
	)(handler)

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{
				Name: "test",
			},
		)

		firstDone <- err
	}()

	select {
	case <-firstBackoffEntered:
	case <-time.After(time.Second):
		t.Fatal("first retry did not enter backoff")
	}

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{
				Name: "test",
			},
		)

		secondDone <- err
	}()

	// The second invocation must wait for the per-tool retry budget.
	select {
	case <-secondBackoffEntered:
		t.Fatal(
			"second retry entered backoff while per-tool budget was occupied",
		)

	case <-time.After(50 * time.Millisecond):
		// Expected.
	}

	close(releaseFirstBackoff)

	select {
	case <-secondBackoffEntered:
		// Expected.
	case <-time.After(time.Second):
		t.Fatal("second retry did not acquire the per-tool budget")
	}

	select {
	case err := <-firstDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"first invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first invocation did not complete")
	}

	select {
	case err := <-secondDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"second invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second invocation did not complete")
	}
}

func TestToolRetryBudget_LimitsGlobalRetries(t *testing.T) {
	testErr := errors.New("temporary failure")

	firstBackoffEntered := make(chan struct{})
	releaseFirstBackoff := make(chan struct{})
	secondBackoffEntered := make(chan struct{})

	var mu sync.Mutex
	backoffCalls := 0

	backoff := func(attempt int) time.Duration {
		mu.Lock()
		backoffCalls++
		call := backoffCalls
		mu.Unlock()

		if call == 1 {
			close(firstBackoffEntered)

			select {
			case <-releaseFirstBackoff:
			case <-time.After(time.Second):
				t.Fatal("first backoff was not released")
			}
		}

		if call == 2 {
			close(secondBackoffEntered)
		}

		return 0
	}

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			return nil, testErr
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,

			ShouldRetry: func(err error) bool {
				return errors.Is(err, testErr)
			},

			Backoff: backoff,

			Budget: &ToolRetryBudgetPolicy{
				Global: 1,
			},
		},
	)(handler)

	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{
				Name: "tool_a",
			},
		)

		firstDone <- err
	}()

	select {
	case <-firstBackoffEntered:
	case <-time.After(time.Second):
		t.Fatal("first retry did not enter backoff")
	}

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{
				Name: "tool_b",
			},
		)

		secondDone <- err
	}()

	// The second invocation must wait for the global retry budget.
	select {
	case <-secondBackoffEntered:
		t.Fatal(
			"second retry entered backoff while global budget was occupied",
		)

	case <-time.After(50 * time.Millisecond):
		// Expected.
	}

	close(releaseFirstBackoff)

	select {
	case <-secondBackoffEntered:
		// Expected.
	case <-time.After(time.Second):
		t.Fatal("second retry did not acquire the global budget")
	}

	select {
	case err := <-firstDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"first invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first invocation did not complete")
	}

	select {
	case err := <-secondDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"second invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second invocation did not complete")
	}
}

func TestToolRetryBudget_CancelWhileWaiting(t *testing.T) {
	testErr := errors.New("temporary failure")

	firstBackoffEntered := make(chan struct{})
	releaseFirstBackoff := make(chan struct{})
	secondBackoffEntered := make(chan struct{})

	var mu sync.Mutex
	backoffCalls := 0

	backoff := func(attempt int) time.Duration {
		mu.Lock()
		backoffCalls++
		call := backoffCalls
		mu.Unlock()

		switch call {
		case 1:
			close(firstBackoffEntered)

			select {
			case <-releaseFirstBackoff:
			case <-time.After(time.Second):
				t.Fatal("first backoff was not released")
			}

		case 2:
			close(secondBackoffEntered)
		}

		return 0
	}

	handler := Handler(
		func(
			ctx context.Context,
			invocation ToolInvocation,
		) (*llm.ToolResult, error) {
			return nil, testErr
		},
	)

	wrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,

			ShouldRetry: func(err error) bool {
				return errors.Is(err, testErr)
			},

			Backoff: backoff,

			Budget: &ToolRetryBudgetPolicy{
				Global: 1,
			},
		},
	)(handler)

	firstDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			context.Background(),
			ToolInvocation{
				Name: "tool_a",
			},
		)

		firstDone <- err
	}()

	select {
	case <-firstBackoffEntered:
	case <-time.After(time.Second):
		t.Fatal("first retry did not enter backoff")
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	secondDone := make(chan error, 1)

	go func() {
		_, err := wrapped(
			ctx,
			ToolInvocation{
				Name: "tool_b",
			},
		)

		secondDone <- err
	}()

	// Give the second invocation a chance to reach the budget.
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
		t.Fatal("cancelled retry did not terminate")
	}

	// Release the first retry.
	close(releaseFirstBackoff)

	select {
	case err := <-firstDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"first invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("first invocation did not complete")
	}

	// Verify the cancelled waiter did not leak the global budget.
	thirdBackoffEntered := make(chan struct{})

	thirdBackoff := func(attempt int) time.Duration {
		close(thirdBackoffEntered)
		return 0
	}

	thirdWrapped := ToolRetry(
		ToolRetryPolicy{
			MaxAttempts: 2,

			ShouldRetry: func(err error) bool {
				return errors.Is(err, testErr)
			},

			Backoff: thirdBackoff,

			Budget: &ToolRetryBudgetPolicy{
				Global: 1,
			},
		},
	)(handler)

	thirdDone := make(chan error, 1)

	go func() {
		_, err := thirdWrapped(
			context.Background(),
			ToolInvocation{
				Name: "tool_c",
			},
		)

		thirdDone <- err
	}()

	select {
	case <-thirdBackoffEntered:
		// Expected: budget was released correctly.
	case <-time.After(time.Second):
		t.Fatal(
			"retry budget remained occupied after cancelled waiter",
		)
	}

	select {
	case err := <-thirdDone:
		if !errors.Is(err, testErr) {
			t.Fatalf(
				"third invocation: expected test error, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("third invocation did not complete")
	}
}
