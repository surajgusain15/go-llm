package tools

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"go-llm/internal/llm"
)

type blockingTestTool struct {
	name    string
	started chan struct{}
}

func (t *blockingTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.name,
			Description: "blocking test tool",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *blockingTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	close(t.started)

	<-ctx.Done()

	return nil, ctx.Err()
}

func TestExecutor_Execute_AppliesToolTimeout(
	t *testing.T,
) {
	registry := NewRegistry()

	started := make(chan struct{})

	registry.Register(
		&blockingTestTool{
			name:    "blocking",
			started: started,
		},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithDefaultToolTimeout(5*time.Second),
		WithToolOutputLimit(64*1024),
		WithMaxToolConcurrency(4),
	)

	start := time.Now()

	_, err := executor.Execute(
		context.Background(),
		"blocking",
		json.RawMessage(`{}`),
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
	case <-started:
		// Tool actually executed.
	default:
		t.Fatal("tool did not execute")
	}

	if elapsed < 50*time.Millisecond {
		t.Fatalf(
			"executor returned before configured timeout: %v",
			elapsed,
		)
	}
}

type largeOutputTestTool struct{}

func (t *largeOutputTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "large_output",
			Description: "returns a large result",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *largeOutputTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {
	return &llm.ToolResult{
		Content: "this output is definitely larger than ten bytes",
	}, nil
}

func TestExecutor_Execute_AppliesToolOutputLimit(
	t *testing.T,
) {
	registry := NewRegistry()

	registry.Register(
		&largeOutputTestTool{},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithToolOutputLimit(10),
	)

	result, err := executor.Execute(
		context.Background(),
		"large_output",
		json.RawMessage(`{}`),
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal("expected tool result")
	}

	content, ok := result.Content.(map[string]any)
	if !ok {
		t.Fatalf(
			"expected truncation metadata object, got %T",
			result.Content,
		)
	}

	if content["truncated"] != true {
		t.Fatalf(
			"expected truncated=true, got %v",
			content["truncated"],
		)
	}

	if content["max_bytes"] != 10 {
		t.Fatalf(
			"expected max_bytes=10, got %v",
			content["max_bytes"],
		)
	}
}

type concurrencyTestTool struct {
	mu sync.Mutex

	current int
	max     int

	started chan struct{}
	release chan struct{}
}

func (t *concurrencyTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "concurrency_test",
			Description: "test tool",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *concurrencyTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	t.mu.Lock()

	t.current++

	if t.current > t.max {
		t.max = t.current
	}

	t.mu.Unlock()

	t.started <- struct{}{}

	select {
	case <-t.release:
	case <-ctx.Done():
		t.finish()
		return nil, ctx.Err()
	}

	t.finish()

	return &llm.ToolResult{
		Content: "ok",
	}, nil
}

func (t *concurrencyTestTool) finish() {
	t.mu.Lock()
	t.current--
	t.mu.Unlock()
}

func (t *concurrencyTestTool) maxConcurrent() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.max
}

func TestExecutor_MaxToolConcurrency(
	t *testing.T,
) {
	registry := NewRegistry()

	tool := &concurrencyTestTool{
		started: make(chan struct{}, 10),
		release: make(chan struct{}),
	}

	registry.Register(tool)

	executor := NewExecutor(
		registry,
		nil,
		WithMaxToolConcurrency(2),
	)

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := executor.Execute(
				context.Background(),
				"concurrency_test",
				json.RawMessage(`{}`),
			)

			if err != nil {
				t.Errorf(
					"unexpected error: %v",
					err,
				)
			}
		}()
	}

	// Give the first two executions time to enter.
	select {
	case <-tool.started:
	case <-time.After(time.Second):
		t.Fatal("first tool execution did not start")
	}

	select {
	case <-tool.started:
	case <-time.After(time.Second):
		t.Fatal("second tool execution did not start")
	}

	// No third execution should be able to start.
	select {
	case <-tool.started:
		t.Fatal(
			"expected maximum 2 concurrent executions",
		)

	case <-time.After(50 * time.Millisecond):
	}

	close(tool.release)

	wg.Wait()

	if tool.maxConcurrent() > 2 {
		t.Fatalf(
			"expected max concurrency <= 2, got %d",
			tool.maxConcurrent(),
		)
	}
}

type blockingExecutorTestTool struct {
	name    string
	started chan struct{}
	release chan struct{}
}

func (t *blockingExecutorTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.name,
			Description: "blocking test tool",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *blockingExecutorTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	close(t.started)

	select {
	case <-t.release:
		return &llm.ToolResult{
			Content: "ok",
		}, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestExecutor_MaxToolConcurrency_CancelWhileWaiting(
	t *testing.T,
) {
	registry := NewRegistry()

	started := make(chan struct{})
	release := make(chan struct{})

	registry.Register(
		&blockingExecutorTestTool{
			name:    "blocking",
			started: started,
			release: release,
		},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithMaxToolConcurrency(1),
	)

	firstDone := make(chan error, 1)

	go func() {
		_, err := executor.Execute(
			context.Background(),
			"blocking",
			json.RawMessage(`{}`),
		)

		firstDone <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first tool did not start")
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	secondDone := make(chan error, 1)

	go func() {
		_, err := executor.Execute(
			ctx,
			"blocking",
			json.RawMessage(`{}`),
		)

		secondDone <- err
	}()

	// Give the second execution a chance to reach
	// the concurrency limiter.
	select {
	case err := <-secondDone:
		t.Fatalf(
			"second tool completed before cancellation: %v",
			err,
		)
	case <-time.After(20 * time.Millisecond):
	}

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
			"waiting tool did not terminate after cancellation",
		)
	}

	close(release)

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

type errorThenSuccessTestTool struct {
	mu sync.Mutex

	name string

	firstStarted  chan struct{}
	secondStarted chan struct{}

	executions int
}

func (t *errorThenSuccessTestTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        t.name,
			Description: "error then success test tool",
			Parameters: llm.ToolParameters{
				Type:       "object",
				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (t *errorThenSuccessTestTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	t.mu.Lock()
	t.executions++
	execution := t.executions
	t.mu.Unlock()

	if execution == 1 {
		close(t.firstStarted)

		return nil, errors.New(
			"intentional test failure",
		)
	}

	close(t.secondStarted)

	return &llm.ToolResult{
		Content: "ok",
	}, nil
}

func TestExecutor_MaxToolConcurrency_SlotReleasedAfterError(
	t *testing.T,
) {
	registry := NewRegistry()

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})

	registry.Register(
		&errorThenSuccessTestTool{
			name:          "test",
			firstStarted:  firstStarted,
			secondStarted: secondStarted,
		},
	)

	executor := NewExecutor(
		registry,
		nil,
		WithMaxToolConcurrency(1),
	)

	firstDone := make(chan error, 1)

	go func() {
		_, err := executor.Execute(
			context.Background(),
			"test",
			json.RawMessage(`{}`),
		)

		firstDone <- err
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first execution did not start")
	}

	select {
	case err := <-firstDone:
		if err == nil {
			t.Fatal("expected first execution to fail")
		}

	case <-time.After(time.Second):
		t.Fatal("first execution did not finish")
	}

	secondDone := make(chan error, 1)

	go func() {
		_, err := executor.Execute(
			context.Background(),
			"test",
			json.RawMessage(`{}`),
		)

		secondDone <- err
	}()

	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal(
			"second execution did not start; semaphore slot may be leaked",
		)
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf(
				"second execution returned unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("second execution did not finish")
	}
}

func TestExecutor_MaxToolConcurrency_NeverExceedsLimit(
	t *testing.T,
) {
	const (
		maxConcurrency = 3
		executions     = 20
	)

	registry := NewRegistry()

	tool := &concurrencyTestTool{
		started: make(chan struct{}, executions),
		release: make(chan struct{}),
	}

	registry.Register(tool)

	executor := NewExecutor(
		registry,
		nil,
		WithMaxToolConcurrency(maxConcurrency),
	)

	var wg sync.WaitGroup

	wg.Add(executions)

	for i := 0; i < executions; i++ {
		go func() {
			defer wg.Done()

			_, err := executor.Execute(
				context.Background(),
				"concurrency_test",
				json.RawMessage(`{}`),
			)

			if err != nil {
				t.Errorf(
					"unexpected execution error: %v",
					err,
				)
			}
		}()
	}

	// Exactly maxConcurrency executions should be able
	// to enter the tool.
	for i := 0; i < maxConcurrency; i++ {
		select {
		case <-tool.started:
		case <-time.After(time.Second):
			t.Fatalf(
				"expected %d tools to start",
				maxConcurrency,
			)
		}
	}

	// No additional execution should enter.
	select {
	case <-tool.started:
		t.Fatal(
			"concurrency limit was exceeded",
		)

	case <-time.After(50 * time.Millisecond):
	}

	close(tool.release)

	wg.Wait()

	if got := tool.maxConcurrent(); got > maxConcurrency {
		t.Fatalf(
			"maximum concurrency was %d, limit was %d",
			got,
			maxConcurrency,
		)
	}
}
