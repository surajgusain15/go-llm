package database

import (
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

type SQLValidator struct {
	parser *sqlparser.Parser
}

func NewSQLValidator() *SQLValidator {
	return &SQLValidator{
		parser: sqlparser.NewTestParser(),
	}
}

func (v *SQLValidator) Validate(query string) error {
	query = strings.TrimSpace(query)

	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	statements, err := v.parser.ParseMultiple(query)
	if err != nil {
		return fmt.Errorf(
			"invalid SQL: %w",
			err,
		)
	}

	if len(statements) != 1 {
		return fmt.Errorf(
			"multiple SQL statements are not allowed",
		)
	}

	statementType := sqlparser.ASTToStatementType(
		statements[0],
	)

	if statementType != sqlparser.StmtSelect {
		return fmt.Errorf(
			"only SELECT queries are allowed",
		)
	}

	return nil
}
