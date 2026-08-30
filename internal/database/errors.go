package database

import (
	"context"
	"errors"
)

type QueryErrorKind string

const (
	QueryErrorNone      QueryErrorKind = ""
	QueryErrorTimeout   QueryErrorKind = "timeout"
	QueryErrorCancelled QueryErrorKind = "cancelled"
	QueryErrorDatabase  QueryErrorKind = "database"
)

func classifyQueryError(
	ctx context.Context,
	err error,
) QueryErrorKind {

	if err == nil {
		return QueryErrorNone
	}

	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return QueryErrorTimeout
	}

	if errors.Is(err, context.Canceled) ||
		errors.Is(ctx.Err(), context.Canceled) {
		return QueryErrorCancelled
	}

	return QueryErrorDatabase
}
