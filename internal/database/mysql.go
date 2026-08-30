package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/sync/singleflight"
)

type schemaCache struct {
	mu        sync.RWMutex
	schema    *SchemaInfo
	expiresAt time.Time
}

type MySQLConfig struct {
	DSN string

	QueryTimeout   time.Duration
	MaxRows        int
	MaxResultBytes int

	SchemaTTL time.Duration

	MaxJoins         int
	MaxUnionBranches int
	MaxSubqueryDepth int

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type DBStats struct {
	MaxOpenConnections int

	OpenConnections int
	InUse           int
	Idle            int

	WaitCount    int64
	WaitDuration time.Duration

	MaxIdleClosed     int64
	MaxIdleTimeClosed int64
	MaxLifetimeClosed int64
}

type MySQLClient struct {
	db               *sql.DB
	queryTimeout     time.Duration
	maxRows          int
	maxResultBytes   int
	maxUnionBranches int
	maxSubqueryDepth int

	validator *SQLValidator

	schemaFlight singleflight.Group

	schemaCache schemaCache
	schemaTTL   time.Duration

	core *core.Core
}

func NewMySQLClient(
	cfg MySQLConfig,
	rt *core.Core,
) (*MySQLClient, error) {

	if rt == nil {
		rt = core.New(nil)
	}

	db, err := sql.Open(
		"mysql",
		cfg.DSN,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"open mysql connection: %w",
			err,
		)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return &MySQLClient{
		db:             db,
		queryTimeout:   cfg.QueryTimeout,
		maxRows:        cfg.MaxRows,
		maxResultBytes: cfg.MaxResultBytes,

		validator: NewSQLValidator(
			cfg.MaxJoins,
			cfg.MaxUnionBranches,
			cfg.MaxSubqueryDepth,
		),

		schemaTTL: cfg.SchemaTTL,

		core: rt,
	}, nil
}

func (c *MySQLClient) Ping(
	ctx context.Context,
) error {

	if ctx == nil {
		ctx = context.Background()
	}

	pingCtx, cancel := context.WithTimeout(
		ctx,
		c.queryTimeout,
	)
	defer cancel()

	if err := c.db.PingContext(pingCtx); err != nil {
		return fmt.Errorf(
			"ping mysql: %w",
			err,
		)
	}

	return nil
}

func (c *MySQLClient) Close() error {
	if c == nil || c.db == nil {
		return nil
	}

	if err := c.db.Close(); err != nil {
		return fmt.Errorf(
			"close mysql connection pool: %w",
			err,
		)
	}

	return nil
}

func (c *MySQLClient) Query(
	ctx context.Context,
	query string,
	args ...any,
) (result *QueryResult, err error) {

	if err = c.validator.Validate(query); err != nil {
		return nil, err
	}

	queryCtx, cancel := context.WithTimeout(
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

		errorKind := classifyQueryError(
			queryCtx,
			err,
		)

		c.core.Emit(
			events.NewDatabaseQueryFinished(
				fingerprint,
				time.Since(start),
				rows,
				err,
				errorKind == QueryErrorTimeout,
				errorKind == QueryErrorCancelled,
			),
		)
	}()

	rows, err := c.db.QueryContext(
		queryCtx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"execute mysql query: %w",
			normalizeQueryError(queryCtx, err),
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
			normalizeQueryError(queryCtx, err),
		)
	}

	result.Count = len(result.Rows)

	return result, nil
}

func normalizeQueryError(
	ctx context.Context,
	err error,
) error {

	if err == nil {
		return nil
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}

	return err
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

	type result struct {
		schema *SchemaInfo
		err    error
	}

	resultCh := make(chan result, 1)

	go func() {

		value, err, _ := c.schemaFlight.Do(
			"schema",
			func() (any, error) {

				// Re-check after entering singleflight.
				// Another request may have refreshed the cache
				// while this request was waiting.
				now := time.Now()

				c.schemaCache.mu.RLock()

				if c.schemaCache.schema != nil &&
					now.Before(c.schemaCache.expiresAt) {

					schema := c.schemaCache.schema

					c.schemaCache.mu.RUnlock()

					return schema, nil
				}

				c.schemaCache.mu.RUnlock()

				// IMPORTANT:
				// Do not use the caller's context here.
				// The refresh belongs to the cache, not to
				// an individual request.
				refreshCtx, cancel := context.WithTimeout(
					context.Background(),
					c.queryTimeout,
				)
				defer cancel()

				return c.refreshSchema(refreshCtx)
			},
		)

		if err != nil {
			resultCh <- result{
				err: err,
			}
			return
		}

		schema, ok := value.(*SchemaInfo)
		if !ok {
			resultCh <- result{
				err: fmt.Errorf(
					"invalid schema cache result type %T",
					value,
				),
			}
			return
		}

		resultCh <- result{
			schema: schema,
		}
	}()

	select {

	case result := <-resultCh:

		if result.err != nil {
			return nil, result.err
		}

		return result.schema, nil

	case <-ctx.Done():

		return nil, ctx.Err()
	}
}

func (c *MySQLClient) refreshSchema(
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

	queryCtx, cancel := context.WithTimeout(
		ctx,
		c.queryTimeout,
	)
	defer cancel()

	rows, err := c.db.QueryContext(
		queryCtx,
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

func (c *MySQLClient) Stats() DBStats {
	stats := c.db.Stats()

	return DBStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}
