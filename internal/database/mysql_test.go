package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMySQLClient(t *testing.T) {

	dsn := os.Getenv("MYSQL_DSN")

	if dsn == "" {
		t.Skip("MYSQL_DSN not set")
	}

	client, err := NewMySQLClient(dsn)
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
