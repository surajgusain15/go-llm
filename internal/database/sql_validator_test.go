package database

import (
	"testing"

	"vitess.io/vitess/go/vt/sqlparser"
)

func TestSQLValidator(t *testing.T) {
	validator := NewSQLValidator(testMaxJoins, testMaxUnionBranches, testMaxSubqueryDepth)

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
		{
			name: "two joins",
			query: `
		SELECT
			t.id,
			p.name,
			s.name
		FROM transactions t
		JOIN providers p
			ON p.id = t.provider_id
		JOIN services s
			ON s.id = t.service_id
	`,
		},
		{
			name: "too many joins",
			query: `
		SELECT *
		FROM transactions t
		JOIN providers p
			ON p.id = t.provider_id
		JOIN services s
			ON s.id = t.service_id
		JOIN provider_services ps
			ON ps.provider_id = p.id
		JOIN transactions t2
			ON t2.id = ps.id
	`,
			shouldErr: true,
		},
		{
			name: "implicit cross join",
			query: `
		SELECT *
		FROM transactions, providers
	`,
			shouldErr: true,
		},
		{
			name: "multiple tables with join",
			query: `
		SELECT t.id, p.name
		FROM transactions t
		JOIN providers p
			ON p.id = t.provider_id
	`,
		},
		{
			name: "one subquery",
			query: `
		SELECT *
		FROM transactions
		WHERE provider_id IN (
			SELECT id
			FROM providers
		)
	`,
		},
		{
			name: "nested subqueries within limit",
			query: `
		SELECT *
		FROM transactions
		WHERE provider_id IN (
			SELECT id
			FROM providers
			WHERE id IN (
				SELECT provider_id
				FROM transaction_attempts
			)
		)
	`,
		},
		{
			name: "five union branches",
			query: `
		SELECT id FROM transactions
		UNION
		SELECT id FROM providers
		UNION
		SELECT id FROM services
		UNION
		SELECT id FROM transaction_attempts
		UNION
		SELECT id FROM provider_services
	`,
		},
		{
			name: "six union branches",
			query: `
		SELECT id FROM transactions
		UNION
		SELECT id FROM providers
		UNION
		SELECT id FROM services
		UNION
		SELECT id FROM transaction_attempts
		UNION
		SELECT id FROM provider_services
		UNION
		SELECT id FROM transactions
	`,
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

func TestDebugCrossJoinAST(t *testing.T) {
	parser := sqlparser.NewTestParser()

	stmt, err := parser.Parse(
		`
		SELECT *
		FROM transactions
		CROSS JOIN providers
	`,
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("statement type: %T", stmt)

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		t.Fatalf("expected *sqlparser.Select, got %T", stmt)
	}

	for i, expr := range selectStmt.From {
		t.Logf("FROM[%d]: %T", i, expr)

		if join, ok := expr.(*sqlparser.JoinTableExpr); ok {
			t.Logf(
				"JOIN[%d]: type=%d typeString=%q condition=%#v",
				i,
				join.Join,
				join.Join.ToString(),
				join.Condition,
			)

			t.Logf(
				"LEFT: %T",
				join.LeftExpr,
			)

			t.Logf(
				"RIGHT: %T",
				join.RightExpr,
			)
		}
	}
}
