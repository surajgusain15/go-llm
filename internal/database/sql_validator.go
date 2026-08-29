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

	stmt := statements[0]

	if sqlparser.ASTToStatementType(stmt) != sqlparser.StmtSelect {
		return fmt.Errorf(
			"only SELECT queries are allowed",
		)
	}

	switch stmt := stmt.(type) {

	case *sqlparser.Select:
		return validateSelect(stmt)

	case *sqlparser.Union:
		return validateUnion(stmt)

	default:
		return fmt.Errorf(
			"unsupported SELECT statement type: %T",
			stmt,
		)
	}
}

func validateSelect(
	stmt *sqlparser.Select,
) error {

	if stmt.Lock != sqlparser.NoLock {
		return fmt.Errorf(
			"locking SELECT queries are not allowed",
		)
	}

	return nil
}

func validateUnion(
	stmt *sqlparser.Union,
) error {

	if stmt.GetLock() != sqlparser.NoLock {
		return fmt.Errorf(
			"locking SELECT queries are not allowed",
		)
	}

	if err := validateTableStatement(stmt.Left); err != nil {
		return err
	}

	if err := validateTableStatement(stmt.Right); err != nil {
		return err
	}

	return nil
}

func validateTableStatement(
	stmt sqlparser.TableStatement,
) error {

	switch stmt := stmt.(type) {

	case *sqlparser.Select:
		return validateSelect(stmt)

	case *sqlparser.Union:
		return validateUnion(stmt)

	default:
		return fmt.Errorf(
			"unsupported UNION statement type: %T",
			stmt,
		)
	}
}
