package database

import "context"

type Client interface {
	Query(
		ctx context.Context,
		query string,
		args ...any,
	) (*QueryResult, error)

	Schema(
		ctx context.Context,
	) (*SchemaInfo, error)

	Ping(
		ctx context.Context,
	) error

	Close() error
}
