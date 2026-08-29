package database

import "testing"

func TestSQLValidator(t *testing.T) {
	validator := NewSQLValidator()

	tests := []struct {
		name      string
		query     string
		shouldErr bool
	}{
		{
			name:  "simple select",
			query: "SELECT 1",
		},
		{
			name:  "select from table",
			query: "SELECT id, status FROM transactions",
		},
		{
			name:  "select with where",
			query: "SELECT id FROM transactions WHERE status = 'SUCCESS'",
		},
		{
			name: "select with group by",
			query: `
				SELECT provider, COUNT(*)
				FROM transactions
				GROUP BY provider
			`,
		},
		{
			name: "select with join",
			query: `
				SELECT t.id, p.name
				FROM transactions t
				JOIN providers p ON p.id = t.provider_id
			`,
		},
		{
			name: "select with subquery",
			query: `
				SELECT *
				FROM transactions
				WHERE provider_id IN (
					SELECT id FROM providers
				)
			`,
		},
		{
			name: "select with union",
			query: `
				SELECT id FROM transactions
				UNION
				SELECT id FROM archived_transactions
			`,
		},
		{
			name:      "empty query",
			query:     "",
			shouldErr: true,
		},
		{
			name:      "whitespace only",
			query:     "   ",
			shouldErr: true,
		},
		{
			name:      "insert",
			query:     "INSERT INTO transactions (status) VALUES ('SUCCESS')",
			shouldErr: true,
		},
		{
			name:      "update",
			query:     "UPDATE transactions SET status = 'SUCCESS'",
			shouldErr: true,
		},
		{
			name:      "delete",
			query:     "DELETE FROM transactions",
			shouldErr: true,
		},
		{
			name:      "drop",
			query:     "DROP TABLE transactions",
			shouldErr: true,
		},
		{
			name:      "alter",
			query:     "ALTER TABLE transactions ADD COLUMN foo VARCHAR(20)",
			shouldErr: true,
		},
		{
			name:      "create",
			query:     "CREATE TABLE test (id INT)",
			shouldErr: true,
		},
		{
			name:      "truncate",
			query:     "TRUNCATE TABLE transactions",
			shouldErr: true,
		},
		{
			name:      "show",
			query:     "SHOW TABLES",
			shouldErr: true,
		},
		{
			name:      "set",
			query:     "SET @foo = 1",
			shouldErr: true,
		},
		{
			name:      "multiple statements",
			query:     "SELECT 1; SELECT 2;",
			shouldErr: true,
		},
		{
			name:      "select followed by update",
			query:     "SELECT 1; UPDATE transactions SET status = 'SUCCESS'",
			shouldErr: true,
		},
		{
			name:      "malformed sql",
			query:     "SELECT FROM WHERE",
			shouldErr: true,
		},
		{
			name:      "locking select",
			query:     "SELECT * FROM transactions FOR UPDATE",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validator.Validate(tt.query)

				if tt.shouldErr && err == nil {
					t.Fatalf(
						"expected error for query %q",
						tt.query,
					)
				}

				if !tt.shouldErr && err != nil {
					t.Fatalf(
						"unexpected error for query %q: %v",
						tt.query,
						err,
					)
				}
			},
		)
	}
}
