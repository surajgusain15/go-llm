package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"slices"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"
	"go-llm/internal/llm"
	"go-llm/internal/service"
)

type ToolInfo struct {
	Name        string
	Description string
}

type Executor struct {
	registry           *Registry
	middlewares        []ToolMiddleware
	core               *core.Core
	defaultToolTimeout time.Duration
	toolTimeouts       map[string]time.Duration
	maxConcurrency     int
	concurrency        chan struct{}
}

func NewExecutor(
	registry *Registry,
	rt *core.Core,
	options ...ExecutorOption,
) *Executor {

	if rt == nil {
		rt = core.New(events.NopObserver{})
	}

	e := &Executor{
		registry: registry,
		core:     rt,
		middlewares: make(
			[]ToolMiddleware,
			0,
		),
		toolTimeouts: make(
			map[string]time.Duration,
		),
		maxConcurrency: service.DefaultToolConcurrency,
	}

	for _, option := range options {
		option(e)
	}

	if e.maxConcurrency > 0 {
		e.concurrency = make(
			chan struct{},
			e.maxConcurrency,
		)
	}

	if e.defaultToolTimeout > 0 ||
		len(e.toolTimeouts) > 0 {

		e.middlewares = append(
			e.middlewares,
			ToolTimeouts(
				e.defaultToolTimeout,
				e.toolTimeouts,
			),
		)
	}

	return e
}

func (e *Executor) Use(
	middleware ToolMiddleware,
) {
	e.middlewares = append(
		e.middlewares,
		middleware,
	)
}

func (e *Executor) List() []ToolInfo {

	tools := e.registry.Tools()

	result := make(
		[]ToolInfo,
		0,
		len(tools),
	)

	for _, tool := range tools {

		schema := tool.Schema()

		result = append(
			result,
			ToolInfo{
				Name:        schema.Function.Name,
				Description: schema.Function.Description,
			},
		)
	}

	return result
}

func (e *Executor) Schemas() []llm.ToolDefinition {

	tools := e.registry.Tools()

	schemas := make(
		[]llm.ToolDefinition,
		0,
		len(tools),
	)

	for _, tool := range tools {
		schemas = append(
			schemas,
			tool.Schema(),
		)
	}

	return schemas
}

func (e *Executor) CollectInput(
	name string,
	reader *bufio.Reader,
) (json.RawMessage, error) {

	tool, ok := e.registry.Get(name)
	if !ok {
		return nil, ErrToolNotFound
	}

	interactiveTool, ok := tool.(InteractiveTool)
	if !ok {
		return nil, ErrNotInteractive
	}

	return interactiveTool.CollectInput(reader)
}

func (e *Executor) Execute(
	ctx context.Context,
	name string,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	if e.concurrency != nil {
		select {
		case e.concurrency <- struct{}{}:
			defer func() {
				<-e.concurrency
			}()

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	start := time.Now()

	e.core.Emit(
		events.NewToolStarted(name),
	)

	tool, ok := e.registry.Get(name)
	if !ok {

		err := ErrToolNotFound

		e.core.Emit(
			events.NewToolFinished(
				name,
				time.Since(start),
				err,
			),
		)

		return nil, err
	}

	handler := func(
		ctx context.Context,
		invocation ToolInvocation,
	) (*llm.ToolResult, error) {

		return tool.Execute(
			ctx,
			invocation.Input,
		)
	}

	for _, middleware := range slices.Backward(
		e.middlewares,
	) {
		handler = middleware(handler)
	}

	result, err := handler(
		ctx,
		ToolInvocation{
			Name:  name,
			Input: input,
		},
	)

	e.core.Emit(
		events.NewToolFinished(
			name,
			time.Since(start),
			err,
		),
	)

	return result, err
}

type ExecutorOption func(*Executor)

func WithToolTimeout(
	name string,
	timeout time.Duration,
) ExecutorOption {
	return func(e *Executor) {
		if e.toolTimeouts == nil {
			e.toolTimeouts = make(
				map[string]time.Duration,
			)
		}

		e.toolTimeouts[name] = timeout
	}
}

func WithDefaultToolTimeout(
	timeout time.Duration,
) ExecutorOption {
	return func(e *Executor) {
		e.defaultToolTimeout = timeout
	}
}

func WithMaxToolConcurrency(
	maxConcurrency int,
) ExecutorOption {
	return func(e *Executor) {
		e.maxConcurrency = maxConcurrency
	}
}
