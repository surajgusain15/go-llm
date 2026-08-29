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

			Description: "Execute a read-only SQL query against the application MySQL database and return the results.",

			Parameters: llm.ToolParameters{
				Type: "object",

				Required: []string{
					"query",
				},

				Properties: map[string]llm.ToolProperty{
					"query": {
						Type:        "string",
						Description: "A read-only SQL SELECT query.",
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
