package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"

	_ "github.com/go-sql-driver/mysql"
)

type schemaCache struct {
	mu        sync.RWMutex
	schema    *SchemaInfo
	expiresAt time.Time
}

type MySQLClient struct {
	db             *sql.DB
	queryTimeout   time.Duration
	maxRows        int
	maxResultBytes int

	validator *SQLValidator

	schemaCache schemaCache
	schemaTTL   time.Duration

	core *core.Core
}

func NewMySQLClient(
	dsn string,
	queryTimeout time.Duration,
	maxRows int,
	maxResultBytes int,
	schemaTTL time.Duration,
	rt *core.Core,
) (*MySQLClient, error) {

	if rt == nil {
		rt = core.New(nil)
	}

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
		db:             db,
		queryTimeout:   queryTimeout,
		maxRows:        maxRows,
		maxResultBytes: maxResultBytes,
		validator:      NewSQLValidator(),
		schemaTTL:      schemaTTL,
		core:           rt,
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
) (result *QueryResult, err error) {

	if err = c.validator.Validate(query); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(
		ctx,
		c.queryTimeout,
	)
	defer cancel()

	fingerprint := fingerprintSQL(query)
	start := time.Now()

	c.core.Emit(
		events.NewDatabaseQueryStarted(
			fingerprint,
		),
	)

	defer func() {

		rows := 0

		if result != nil {
			rows = result.Count
		}

		c.core.Emit(
			events.NewDatabaseQueryFinished(
				fingerprint,
				time.Since(start),
				rows,
				err,
			),
		)
	}()

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

	var resultBytes int

	result = &QueryResult{
		Columns: columns,
		Rows:    make([][]any, 0),
	}

	for rows.Next() {

		if len(result.Rows) >= c.maxRows {
			result.Truncated = true
			result.TruncateReason = "max_rows"
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

		rowBytes := 0

		for i, value := range values {

			switch v := value.(type) {

			case []byte:
				values[i] = string(v)
				rowBytes += len(v)

			case string:
				rowBytes += len(v)

			default:
				data, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf(
						"marshal mysql value: %w",
						err,
					)
				}

				rowBytes += len(data)
			}
		}

		if resultBytes+rowBytes > c.maxResultBytes {
			result.Truncated = true
			result.TruncateReason = "max_result_bytes"
			break
		}

		result.Rows = append(
			result.Rows,
			values,
		)

		resultBytes += rowBytes
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

	now := time.Now()

	c.schemaCache.mu.RLock()

	if c.schemaCache.schema != nil &&
		now.Before(c.schemaCache.expiresAt) {

		schema := c.schemaCache.schema

		c.schemaCache.mu.RUnlock()

		return schema, nil
	}

	c.schemaCache.mu.RUnlock()

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

	c.schemaCache.mu.Lock()

	c.schemaCache.schema = schema
	c.schemaCache.expiresAt = time.Now().Add(
		c.schemaTTL,
	)

	c.schemaCache.mu.Unlock()

	return schema, nil
}
