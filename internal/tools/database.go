package tools

import (
	"context"
	"fmt"

	"go-llm/internal/database"
	"go-llm/internal/llm"
)

type DatabaseQueryTool struct {
	db database.Client
}

func NewDatabaseQueryTool(
	db database.Client,
) *DatabaseQueryTool {
	return &DatabaseQueryTool{
		db: db,
	}
}

func (t *DatabaseQueryTool) Name() string {
	return "database_query"
}

func (t *DatabaseQueryTool) Description() string {
	return "Execute a read-only SQL query against the application database and return the results."
}

func (t *DatabaseQueryTool) Execute(
	ctx context.Context,
	input map[string]any,
) (*llm.ToolResult, error) {

	queryValue, ok := input["query"]
	if !ok {
		return nil, fmt.Errorf(
			"database_query: missing query",
		)
	}

	query, ok := queryValue.(string)
	if !ok {
		return nil, fmt.Errorf(
			"database_query: query must be a string",
		)
	}

	result, err := t.db.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}

	return &llm.ToolResult{
		Content: result,
	}, nil
}
