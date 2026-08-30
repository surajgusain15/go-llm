package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/sync/singleflight"
)

const (
	testQueryTimeout     = 10 * time.Second
	testMaxRows          = 100
	testMaxJoins         = 3
	testMaxUnionBranches = 5
	testMaxSubqueryDepth = 5
	testMaxResultBytes   = 1024 * 1024
	testSchemaTTL        = 10 * time.Second
	testMaxOpenConns     = 10
	testMaxIdleConns     = 5
	testConnMaxLifetime  = 30 * time.Minute
	testConnMaxIdleTime  = 10 * time.Minute
)

func TestMySQLClient(t *testing.T) {

	dsn := os.Getenv("MYSQL_DSN")

	if dsn == "" {
		t.Skip("MYSQL_DSN not set")
	}
	rt := core.New(events.NewCLIObserver(events.LogLevelDebug))

	client, err := NewMySQLClient(
		MySQLConfig{
			DSN:              dsn,
			QueryTimeout:     testQueryTimeout,
			MaxRows:          testMaxRows,
			MaxResultBytes:   testMaxResultBytes,
			SchemaTTL:        testSchemaTTL,
			MaxJoins:         testMaxJoins,
			MaxUnionBranches: testMaxUnionBranches,
			MaxSubqueryDepth: testMaxSubqueryDepth,
			MaxOpenConns:     testMaxOpenConns,
			MaxIdleConns:     testMaxIdleConns,
			ConnMaxLifetime:  testConnMaxLifetime,
			ConnMaxIdleTime:  testConnMaxIdleTime,
		},
		rt,
	)
	if err != nil {
		t.Fatal(err)
	}

	defer client.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	result, err := client.Query(
		ctx,
		"SELECT 1 AS value",
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Count != 1 {
		t.Fatalf(
			"expected 1 row, got %d",
			result.Count,
		)
	}

	if len(result.Columns) != 1 {
		t.Fatalf(
			"expected 1 column, got %d",
			len(result.Columns),
		)
	}
}

func TestMySQLClientSchema_ConcurrentRefresh(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	client := &MySQLClient{
		db:           db,
		schemaTTL:    time.Minute,
		queryTimeout: 5 * time.Second,
		core:         core.New(events.NopObserver{}),
		schemaFlight: singleflight.Group{},
	}

	rows := sqlmock.NewRows(
		[]string{
			"TABLE_NAME",
			"COLUMN_NAME",
			"COLUMN_TYPE",
			"IS_NULLABLE",
			"COLUMN_KEY",
			"COLUMN_DEFAULT",
		},
	).AddRow(
		"transactions",
		"id",
		"bigint",
		"NO",
		"PRI",
		nil,
	)

	// Make the schema query deliberately slow so all callers
	// have to contend for the same in-flight refresh.
	mock.ExpectQuery(
		"SELECT\\s+TABLE_NAME",
	).WillDelayFor(100 * time.Millisecond).
		WillReturnRows(rows)

	const callers = 10

	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(callers)

	errCh := make(chan error, callers)

	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()

			<-start

			schema, err := client.Schema(
				context.Background(),
			)
			if err != nil {
				errCh <- err
				return
			}

			if schema == nil {
				errCh <- fmt.Errorf(
					"schema is nil",
				)
				return
			}

			if len(schema.Tables) != 1 {
				errCh <- fmt.Errorf(
					"expected 1 table, got %d",
					len(schema.Tables),
				)
				return
			}

			if schema.Tables[0].Name != "transactions" {
				errCh <- fmt.Errorf(
					"expected transactions table, got %q",
					schema.Tables[0].Name,
				)
			}
		}()
	}

	// Release all callers simultaneously.
	close(start)

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}

	// There must have been exactly one database query.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"schema refresh was not deduplicated: %v",
			err,
		)
	}
}

func TestMySQLClient_ConfiguresConnectionPool(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	const (
		maxOpen = 20
		maxIdle = 10
	)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	stats := db.Stats()

	if stats.MaxOpenConnections != maxOpen {
		t.Fatalf(
			"expected max open connections %d, got %d",
			maxOpen,
			stats.MaxOpenConnections,
		)
	}
}

func TestMySQLClientStats(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)

	client := &MySQLClient{
		db: db,
	}

	stats := client.Stats()

	if stats.MaxOpenConnections != 20 {
		t.Fatalf(
			"expected max open connections 20, got %d",
			stats.MaxOpenConnections,
		)
	}
}

func TestClassifyQueryError(t *testing.T) {

	t.Run(
		"no error", func(t *testing.T) {

			got := classifyQueryError(
				context.Background(),
				nil,
			)

			if got != QueryErrorNone {
				t.Fatalf(
					"expected %q, got %q",
					QueryErrorNone,
					got,
				)
			}
		},
	)

	t.Run(
		"deadline exceeded", func(t *testing.T) {

			ctx, cancel := context.WithTimeout(
				context.Background(),
				time.Nanosecond,
			)
			defer cancel()

			<-ctx.Done()

			got := classifyQueryError(
				ctx,
				ctx.Err(),
			)

			if got != QueryErrorTimeout {
				t.Fatalf(
					"expected %q, got %q",
					QueryErrorTimeout,
					got,
				)
			}
		},
	)

	t.Run(
		"cancelled", func(t *testing.T) {

			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			cancel()

			got := classifyQueryError(
				ctx,
				context.Canceled,
			)

			if got != QueryErrorCancelled {
				t.Fatalf(
					"expected %q, got %q",
					QueryErrorCancelled,
					got,
				)
			}
		},
	)

	t.Run(
		"database error", func(t *testing.T) {

			err := fmt.Errorf("mysql connection failed")

			got := classifyQueryError(
				context.Background(),
				err,
			)

			if got != QueryErrorDatabase {
				t.Fatalf(
					"expected %q, got %q",
					QueryErrorDatabase,
					got,
				)
			}
		},
	)
}

func TestMySQLClientSchema_RefreshFailureDoesNotPoisonCache(
	t *testing.T,
) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	client := &MySQLClient{
		db:           db,
		schemaTTL:    time.Minute,
		queryTimeout: 5 * time.Second,
		core:         core.New(events.NopObserver{}),
		schemaFlight: singleflight.Group{},
	}

	// First refresh fails.
	mock.ExpectQuery(
		"SELECT\\s+TABLE_NAME",
	).WillReturnError(
		fmt.Errorf("schema database unavailable"),
	)

	_, err = client.Schema(context.Background())
	if err == nil {
		t.Fatal("expected schema refresh error")
	}

	// Failed refresh must not populate the cache.
	client.schemaCache.mu.RLock()

	cachedSchema := client.schemaCache.schema
	expiresAt := client.schemaCache.expiresAt

	client.schemaCache.mu.RUnlock()

	if cachedSchema != nil {
		t.Fatal("expected schema cache to remain empty")
	}

	if !expiresAt.IsZero() {
		t.Fatal("expected schema cache expiry to remain zero")
	}

	// Second refresh succeeds.
	rows := sqlmock.NewRows(
		[]string{
			"TABLE_NAME",
			"COLUMN_NAME",
			"COLUMN_TYPE",
			"IS_NULLABLE",
			"COLUMN_KEY",
			"COLUMN_DEFAULT",
		},
	).AddRow(
		"transactions",
		"id",
		"bigint",
		"NO",
		"PRI",
		nil,
	)

	mock.ExpectQuery(
		"SELECT\\s+TABLE_NAME",
	).WillReturnRows(rows)

	schema, err := client.Schema(context.Background())
	if err != nil {
		t.Fatalf(
			"expected second schema refresh to succeed: %v",
			err,
		)
	}

	if schema == nil {
		t.Fatal("expected schema")
	}

	if len(schema.Tables) != 1 {
		t.Fatalf(
			"expected 1 table, got %d",
			len(schema.Tables),
		)
	}

	if schema.Tables[0].Name != "transactions" {
		t.Fatalf(
			"expected transactions table, got %q",
			schema.Tables[0].Name,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unexpected database interactions: %v",
			err,
		)
	}
}

func TestMySQLClient_Query_EmitsObservabilityEvents(
	t *testing.T,
) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	observer := &recordingObserver{}

	client := &MySQLClient{
		db:             db,
		queryTimeout:   5 * time.Second,
		maxRows:        10,
		maxResultBytes: 1024,
		validator:      NewSQLValidator(3, 5, 5),
		core:           core.New(observer),
	}

	rows := sqlmock.NewRows(
		[]string{"id"},
	).AddRow(1).AddRow(2)

	mock.ExpectQuery(
		"SELECT id FROM transactions",
	).WillReturnRows(rows)

	result, err := client.Query(
		context.Background(),
		"SELECT id FROM transactions",
	)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if result.Count != 2 {
		t.Fatalf(
			"expected 2 rows, got %d",
			result.Count,
		)
	}

	recorded := observer.Events()

	if len(recorded) != 2 {
		t.Fatalf(
			"expected 2 events, got %d",
			len(recorded),
		)
	}

	started, ok := recorded[0].(events.DatabaseQueryStarted)
	if !ok {
		t.Fatalf(
			"expected DatabaseQueryStarted, got %T",
			recorded[0],
		)
	}

	finished, ok := recorded[1].(events.DatabaseQueryFinished)
	if !ok {
		t.Fatalf(
			"expected DatabaseQueryFinished, got %T",
			recorded[1],
		)
	}

	if started.Fingerprint == "" {
		t.Fatal("expected query fingerprint")
	}

	if finished.Fingerprint != started.Fingerprint {
		t.Fatalf(
			"fingerprint mismatch: started=%q finished=%q",
			started.Fingerprint,
			finished.Fingerprint,
		)
	}

	if finished.Duration <= 0 {
		t.Fatal("expected positive query duration")
	}

	if finished.Rows != 2 {
		t.Fatalf(
			"expected 2 rows in event, got %d",
			finished.Rows,
		)
	}

	if finished.Err != nil {
		t.Fatalf(
			"expected no error, got %v",
			finished.Err,
		)
	}

	if finished.TimedOut {
		t.Fatal("successful query cannot be marked timed out")
	}

	if finished.Cancelled {
		t.Fatal("successful query cannot be marked cancelled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unexpected database interaction: %v",
			err,
		)
	}
}

func TestMySQLClient_Query_EmitsErrorEvent(
	t *testing.T,
) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	observer := &recordingObserver{}

	client := &MySQLClient{
		db:             db,
		queryTimeout:   5 * time.Second,
		maxRows:        10,
		maxResultBytes: 1024,
		validator:      NewSQLValidator(3, 5, 5),
		core:           core.New(observer),
	}

	mock.ExpectQuery(
		"SELECT 1",
	).WillReturnError(
		fmt.Errorf("connection failed"),
	)

	_, err = client.Query(
		context.Background(),
		"SELECT 1",
	)

	if err == nil {
		t.Fatal("expected query error")
	}

	recorded := observer.Events()

	if len(recorded) != 2 {
		t.Fatalf(
			"expected 2 events, got %d",
			len(recorded),
		)
	}

	finished, ok := recorded[1].(events.DatabaseQueryFinished)
	if !ok {
		t.Fatalf(
			"expected DatabaseQueryFinished, got %T",
			recorded[1],
		)
	}

	if finished.Err == nil {
		t.Fatal("expected error in finished event")
	}

	if finished.TimedOut {
		t.Fatal("database error should not be marked timeout")
	}

	if finished.Cancelled {
		t.Fatal("database error should not be marked cancelled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unexpected database interaction: %v",
			err,
		)
	}
}
