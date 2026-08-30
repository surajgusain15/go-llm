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
)

func TestMySQLClient(t *testing.T) {

	dsn := os.Getenv("MYSQL_DSN")

	if dsn == "" {
		t.Skip("MYSQL_DSN not set")
	}
	rt := core.New(events.NewCLIObserver(events.LogLevelDebug))

	client, err := NewMySQLClient(
		dsn,
		testQueryTimeout,
		testMaxRows,
		testMaxResultBytes,
		testSchemaTTL,
		rt,
		testMaxJoins,
		testMaxUnionBranches,
		testMaxSubqueryDepth,
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
