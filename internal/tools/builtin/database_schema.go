package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"go-llm/internal/database"
	"go-llm/internal/llm"
)

type DatabaseSchemaTool struct {
	db database.Client
}

func NewDatabaseSchemaTool(
	db database.Client,
) *DatabaseSchemaTool {
	return &DatabaseSchemaTool{
		db: db,
	}
}

func (d *DatabaseSchemaTool) Schema() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: llm.ToolTypeFunction,

		Function: llm.ToolFunction{
			Name: "database_schema",

			Description: `Get the current application database schema, including available tables and their columns.

Use this tool before writing SQL whenever the required table or column names are not already known with certainty.

Do not invent column names. Use the returned schema to determine:
- which tables contain the required data
- exact column names
- column types
- primary/key information
- nullable columns

After inspecting the schema, use database_query to execute a read-only SELECT query.`,

			Parameters: llm.ToolParameters{
				Type: "object",

				Properties: map[string]llm.ToolProperty{},
			},
		},
	}
}

func (d *DatabaseSchemaTool) Execute(
	ctx context.Context,
	input json.RawMessage,
) (*llm.ToolResult, error) {

	schema, err := d.db.Schema(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"get database schema: %w",
			err,
		)
	}

	return &llm.ToolResult{
		Content: schema,
	}, nil
}
