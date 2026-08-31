package tools

import (
	"context"

	"go-llm/internal/core"
)

type middlewareContextKey struct{}

type ToolMiddlewareContext struct {
	Core *core.Core
}

func withToolMiddlewareContext(
	ctx context.Context,
	core *core.Core,
) context.Context {

	return context.WithValue(
		ctx,
		middlewareContextKey{},
		ToolMiddlewareContext{
			Core: core,
		},
	)
}

func toolMiddlewareContext(
	ctx context.Context,
) ToolMiddlewareContext {

	value := ctx.Value(middlewareContextKey{})

	if value == nil {
		return ToolMiddlewareContext{}
	}

	middlewareCtx, ok := value.(ToolMiddlewareContext)

	if !ok {
		return ToolMiddlewareContext{}
	}

	return middlewareCtx
}
