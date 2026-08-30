package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"go-llm/internal/database"
	"go-llm/internal/llm"
)

type DatabaseQueryInput struct {
	Query string `json:"query"`
}

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

func (d *DatabaseQueryTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: llm.ToolTypeFunction,

		Function: llm.ToolFunction{
			Name: "database_query",

			Description: `Execute exactly one read-only SQL SELECT query against the application MySQL database.

Rules:
- Only SELECT queries are allowed.
- The query must contain exactly one SQL statement.
- INSERT, UPDATE, DELETE, DDL, SET, SHOW, and other non-SELECT statements are not allowed.
- Locking SELECT queries such as FOR UPDATE are not allowed.
- CROSS JOIN is not allowed.
- Queries with excessive JOINs, UNION branches, or nested subqueries are rejected.
- Use database_schema first when table or column names are unknown.
- Never invent table or column names.
- Prefer targeted queries that return only the columns and rows needed to answer the user's question.
- Query results are server-side limited and may be truncated.`,

			Parameters: llm.ToolParameters{
				Type: "object",

				Required: []string{
					"query",
				},

				Properties: map[string]llm.ToolProperty{
					"query": {
						Type:        "string",
						Description: "One valid read-only SELECT statement using only tables and columns confirmed by database_schema.",
					},
				},
			},
		},
	}
}

func (d *DatabaseQueryTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	var req DatabaseQueryInput

	if err := json.Unmarshal(
		input,
		&req,
	); err != nil {
		return nil, fmt.Errorf(
			"decode database query input: %w",
			err,
		)
	}

	if req.Query == "" {
		return nil, fmt.Errorf(
			"database query cannot be empty",
		)
	}

	if err := database.ValidateReadOnlyQuery(
		req.Query,
	); err != nil {
		return nil, err
	}

	result, err := d.db.Query(
		ctx,
		req.Query,
	)
	if err != nil {
		return nil, err
	}

	return &llm.ToolResult{
		Content: result,
	}, nil
}
