package database

import "testing"

func TestValidateReadOnlyQuery(t *testing.T) {

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:  "select",
			query: "SELECT 1",
		},
		{
			name:  "select with where",
			query: "SELECT id FROM users WHERE id = 1",
		},
		{
			name:    "insert",
			query:   "INSERT INTO users VALUES (1)",
			wantErr: true,
		},
		{
			name:    "update",
			query:   "UPDATE users SET name = 'x'",
			wantErr: true,
		},
		{
			name:    "delete",
			query:   "DELETE FROM users",
			wantErr: true,
		},
		{
			name:    "drop",
			query:   "DROP TABLE users",
			wantErr: true,
		},
		{
			name:    "alter",
			query:   "ALTER TABLE users ADD foo VARCHAR(10)",
			wantErr: true,
		},
		{
			name:    "truncate",
			query:   "TRUNCATE TABLE users",
			wantErr: true,
		},
		{
			name:    "multiple statements",
			query:   "SELECT 1; DROP TABLE users",
			wantErr: true,
		},
		{
			name:    "comment",
			query:   "SELECT 1 -- comment",
			wantErr: true,
		},
		{
			name:    "empty",
			query:   "",
			wantErr: true,
		},
		{
			name:    "not select",
			query:   "SHOW TABLES",
			wantErr: true,
		},
	}

	for _, tt := range tests {

		t.Run(
			tt.name, func(t *testing.T) {

				err := ValidateReadOnlyQuery(
					tt.query,
				)

				if tt.wantErr && err == nil {
					t.Fatalf(
						"expected error for query %q",
						tt.query,
					)
				}

				if !tt.wantErr && err != nil {
					t.Fatalf(
						"unexpected error: %v",
						err,
					)
				}
			},
		)
	}
}
