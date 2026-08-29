package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLClient struct {
	db           *sql.DB
	queryTimeout time.Duration
	maxRows      int
}

func NewMySQLClient(
	dsn string,
	queryTimeout time.Duration,
	maxRows int,
) (*MySQLClient, error) {

	db, err := sql.Open(
		"mysql",
		dsn,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open mysql connection: %w",
			err,
		)
	}

	return &MySQLClient{
		db:           db,
		queryTimeout: queryTimeout,
		maxRows:      maxRows,
	}, nil
}

func (c *MySQLClient) Ping(
	ctx context.Context,
) error {

	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf(
			"ping mysql: %w",
			err,
		)
	}

	return nil
}

func (c *MySQLClient) Close() error {
	return c.db.Close()
}

func (c *MySQLClient) Query(
	ctx context.Context,
	query string,
	args ...any,
) (*QueryResult, error) {

	ctx, cancel := context.WithTimeout(
		ctx,
		c.queryTimeout,
	)
	defer cancel()

	rows, err := c.db.QueryContext(
		ctx,
		query,
		args...,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"execute mysql query: %w",
			err,
		)
	}

	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf(
			"get mysql columns: %w",
			err,
		)
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([][]any, 0),
	}

	for rows.Next() {

		if len(result.Rows) >= c.maxRows {
			result.Truncated = true
			break
		}

		values := make([]any, len(columns))
		scanTargets := make([]any, len(columns))

		for i := range values {
			scanTargets[i] = &values[i]
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf(
				"scan mysql row: %w",
				err,
			)
		}

		for i, value := range values {
			if v, ok := value.([]byte); ok {
				values[i] = string(v)
			}
		}

		result.Rows = append(
			result.Rows,
			values,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate mysql rows: %w",
			err,
		)
	}

	result.Count = len(result.Rows)

	return result, nil
}

func (c *MySQLClient) Schema(
	ctx context.Context,
) (*SchemaInfo, error) {

	const query = `
SELECT
	TABLE_NAME,
	COLUMN_NAME,
	COLUMN_TYPE,
	IS_NULLABLE,
	COLUMN_KEY,
	COLUMN_DEFAULT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
ORDER BY TABLE_NAME, ORDINAL_POSITION
`

	rows, err := c.db.QueryContext(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query database schema: %w",
			err,
		)
	}

	defer rows.Close()

	schema := &SchemaInfo{
		Tables: make([]TableInfo, 0),
	}

	tableIndex := make(map[string]int)

	for rows.Next() {

		var (
			tableName     string
			columnName    string
			columnType    string
			nullable      string
			columnKey     string
			columnDefault any
		)

		if err := rows.Scan(
			&tableName,
			&columnName,
			&columnType,
			&nullable,
			&columnKey,
			&columnDefault,
		); err != nil {
			return nil, fmt.Errorf(
				"scan database schema: %w",
				err,
			)
		}

		index, exists := tableIndex[tableName]

		if !exists {

			index = len(schema.Tables)
			tableIndex[tableName] = index

			schema.Tables = append(
				schema.Tables,
				TableInfo{
					Name: tableName,
				},
			)
		}

		schema.Tables[index].Columns = append(
			schema.Tables[index].Columns,
			ColumnInfo{
				Name:     columnName,
				Type:     columnType,
				Nullable: nullable,
				Key:      columnKey,
				Default:  columnDefault,
			},
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate mysql schema: %w",
			err,
		)
	}

	return schema, nil
}
