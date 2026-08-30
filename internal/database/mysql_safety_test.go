package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-llm/internal/core"
	"go-llm/internal/events"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/sync/singleflight"
)

func newMockMySQLClient(
	t *testing.T,
) (*MySQLClient, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	client := &MySQLClient{
		db:             db,
		queryTimeout:   100 * time.Millisecond,
		maxRows:        2,
		maxResultBytes: 1024,
		validator:      NewSQLValidator(3, 5, 5),
		schemaTTL:      time.Minute,
		schemaFlight:   singleflight.Group{},
		core:           core.New(events.NopObserver{}),
	}

	return client, mock
}

func TestMySQLClient_Query_RejectsUnsafeSQLBeforeExecution(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "insert",
			query: "INSERT INTO transactions (status) VALUES ('SUCCESS')",
		},
		{
			name:  "update",
			query: "UPDATE transactions SET status = 'SUCCESS'",
		},
		{
			name:  "delete",
			query: "DELETE FROM transactions",
		},
		{
			name:  "ddl",
			query: "DROP TABLE transactions",
		},
		{
			name:  "multiple statements",
			query: "SELECT 1; SELECT 2",
		},
		{
			name: "cross join",
			query: `
				SELECT *
				FROM transactions
				CROSS JOIN providers
			`,
		},
		{
			name: "too many joins",
			query: `
				SELECT *
				FROM transactions t
				JOIN providers p ON p.id = t.provider_id
				JOIN services s ON s.id = t.service_id
				JOIN provider_services ps ON ps.provider_id = p.id
				JOIN transaction_attempts ta
					ON ta.transaction_id = t.id
			`,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				_, err := client.Query(
					context.Background(),
					tt.query,
				)

				if err == nil {
					t.Fatalf(
						"expected query to be rejected: %s",
						tt.query,
					)
				}
			},
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unsafe queries reached MySQL: %v",
			err,
		)
	}
}

func TestMySQLClient_Query_ReturnsRowsWithinLimit(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	rows := sqlmock.NewRows(
		[]string{"id", "status"},
	).
		AddRow(1, "SUCCESS").
		AddRow(2, "FAILED").
		AddRow(3, "SUCCESS")

	mock.ExpectQuery(
		"SELECT id, status FROM transactions",
	).WillReturnRows(rows)

	result, err := client.Query(
		context.Background(),
		"SELECT id, status FROM transactions",
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

	if !result.Truncated {
		t.Fatal("expected result to be truncated")
	}

	if result.TruncateReason != "max_rows" {
		t.Fatalf(
			"expected max_rows, got %q",
			result.TruncateReason,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database interaction: %v", err)
	}
}

func TestMySQLClient_Query_ReturnsBytesWithinLimit(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	client.maxRows = 100
	client.maxResultBytes = 10

	rows := sqlmock.NewRows(
		[]string{"value"},
	).
		AddRow("12345").
		AddRow("67890").
		AddRow("abcdefghij")

	mock.ExpectQuery(
		"SELECT value FROM transactions",
	).WillReturnRows(rows)

	result, err := client.Query(
		context.Background(),
		"SELECT value FROM transactions",
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

	if !result.Truncated {
		t.Fatal("expected result to be truncated")
	}

	if result.TruncateReason != "max_result_bytes" {
		t.Fatalf(
			"expected max_result_bytes, got %q",
			result.TruncateReason,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database interaction: %v", err)
	}
}

func TestMySQLClient_Query_PropagatesDatabaseError(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	mock.ExpectQuery(
		"SELECT 1",
	).WillReturnError(
		fmt.Errorf("database unavailable"),
	)

	result, err := client.Query(
		context.Background(),
		"SELECT 1",
	)

	if result != nil {
		t.Fatal("expected nil result on database error")
	}

	if err == nil {
		t.Fatal("expected database error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database interaction: %v", err)
	}
}

func TestMySQLClient_Query_ContextCancellation(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	_, err := client.Query(
		ctx,
		"SELECT 1",
	)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"cancelled query unexpectedly reached MySQL: %v",
			err,
		)
	}
}

func TestMySQLClient_Query_Timeout(
	t *testing.T,
) {
	client, mock := newMockMySQLClient(t)

	client.queryTimeout = 10 * time.Millisecond

	mock.ExpectQuery(
		"SELECT SLEEP",
	).WillDelayFor(
		100 * time.Millisecond,
	).WillReturnRows(
		sqlmock.NewRows([]string{"value"}).
			AddRow(1),
	)

	_, err := client.Query(
		context.Background(),
		"SELECT SLEEP(1)",
	)

	if err == nil {
		t.Fatal("expected query timeout")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"unexpected database interaction: %v",
			err,
		)
	}
}
