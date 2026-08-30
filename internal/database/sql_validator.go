package database

import (
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

type SQLValidator struct {
	parser   *sqlparser.Parser
	maxJoins int
}

func NewSQLValidator(maxJoins int) *SQLValidator {
	return &SQLValidator{
		parser:   sqlparser.NewTestParser(),
		maxJoins: maxJoins,
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
		return v.validateSelect(stmt)

	case *sqlparser.Union:
		return v.validateUnion(stmt)

	default:
		return fmt.Errorf(
			"unsupported SELECT statement type: %T",
			stmt,
		)
	}
}

func (v *SQLValidator) validateSelect(
	stmt *sqlparser.Select,
) error {

	if stmt.GetLock() != sqlparser.NoLock {
		return fmt.Errorf(
			"locking SELECT queries are not allowed",
		)
	}

	joins := countJoins(stmt.From)

	if joins > v.maxJoins {
		return fmt.Errorf(
			"query contains too many joins: %d (maximum %d)",
			joins,
			v.maxJoins,
		)
	}

	if err := validateNoCrossJoins(stmt.From); err != nil {
		return err
	}

	return nil
}

func (v *SQLValidator) validateUnion(
	stmt *sqlparser.Union,
) error {

	if stmt.GetLock() != sqlparser.NoLock {
		return fmt.Errorf(
			"locking SELECT queries are not allowed",
		)
	}

	if err := v.validateTableStatement(stmt.Left); err != nil {
		return err
	}

	if err := v.validateTableStatement(stmt.Right); err != nil {
		return err
	}

	return nil
}

func (v *SQLValidator) validateTableStatement(
	stmt sqlparser.TableStatement,
) error {

	switch stmt := stmt.(type) {

	case *sqlparser.Select:
		return v.validateSelect(stmt)

	case *sqlparser.Union:
		return v.validateUnion(stmt)

	default:
		return fmt.Errorf(
			"unsupported SELECT statement type: %T",
			stmt,
		)
	}
}

func countJoins(
	from sqlparser.TableExprs,
) int {

	count := 0

	for _, expr := range from {
		count += countJoinsInExpr(expr)
	}

	return count
}

func countJoinsInExpr(
	expr sqlparser.TableExpr,
) int {

	switch expr := expr.(type) {

	case *sqlparser.JoinTableExpr:

		return 1 +
			countJoinsInExpr(expr.LeftExpr) +
			countJoinsInExpr(expr.RightExpr)

	case *sqlparser.AliasedTableExpr:

		return 0

	case *sqlparser.ParenTableExpr:

		count := 0

		for _, tableExpr := range expr.Exprs {
			count += countJoinsInExpr(tableExpr)
		}

		return count

	default:
		return 0
	}
}

func validateNoCrossJoins(
	from sqlparser.TableExprs,
) error {

	if len(from) > 1 {
		return fmt.Errorf(
			"implicit cross joins are not allowed",
		)
	}

	for _, expr := range from {
		if err := validateTableExprForCrossJoin(expr); err != nil {
			return err
		}
	}

	return nil
}

func validateTableExprForCrossJoin(
	expr sqlparser.TableExpr,
) error {

	switch expr := expr.(type) {

	case *sqlparser.JoinTableExpr:

		joinSQL := sqlparser.String(expr)

		if strings.Contains(
			strings.ToUpper(joinSQL),
			" CROSS JOIN ",
		) {
			return fmt.Errorf(
				"CROSS JOIN is not allowed",
			)
		}

		if err := validateTableExprForCrossJoin(
			expr.LeftExpr,
		); err != nil {
			return err
		}

		return validateTableExprForCrossJoin(
			expr.RightExpr,
		)

	case *sqlparser.ParenTableExpr:

		for _, tableExpr := range expr.Exprs {
			if err := validateTableExprForCrossJoin(
				tableExpr,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
