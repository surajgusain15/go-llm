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

			Description: "Get the tables and columns available in the application database. Use this before writing SQL when the database structure is unknown.",

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
