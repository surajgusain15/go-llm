package database

import (
	"fmt"
	"strings"
	"unicode"
)

func ValidateReadOnlyQuery(
	query string,
) error {

	query = strings.TrimSpace(query)

	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	// Reject SQL comments.
	if strings.Contains(query, "--") ||
		strings.Contains(query, "/*") ||
		strings.Contains(query, "*/") {
		return fmt.Errorf("SQL comments are not allowed")
	}

	// Reject multiple statements.
	if hasMultipleStatements(query) {
		return fmt.Errorf("multiple SQL statements are not allowed")
	}

	normalized := strings.ToLower(
		strings.TrimSpace(query),
	)

	// Only SELECT is allowed for now.
	if !strings.HasPrefix(normalized, "select") {
		return fmt.Errorf(
			"only SELECT queries are allowed",
		)
	}

	// Make sure "select" is actually a SQL keyword,
	// not something like "selectfoo".
	if len(normalized) > len("select") &&
		!unicode.IsSpace(
			rune(normalized[len("select")]),
		) {
		return fmt.Errorf(
			"query must begin with SELECT",
		)
	}

	blocked := []string{
		"insert",
		"update",
		"delete",
		"drop",
		"alter",
		"truncate",
		"create",
		"replace",
		"grant",
		"revoke",
		"rename",
		"load",
		"outfile",
		"dumpfile",
		"call",
		"set",
		"transaction",
		"commit",
		"rollback",
		"lock",
		"unlock",
	}

	for _, keyword := range blocked {

		if containsSQLKeyword(
			normalized,
			keyword,
		) {
			return fmt.Errorf(
				"SQL operation %q is not allowed",
				keyword,
			)
		}
	}

	return nil
}

func containsSQLKeyword(
	query string,
	keyword string,
) bool {

	for i := 0; i+len(keyword) <= len(query); i++ {

		if query[i:i+len(keyword)] != keyword {
			continue
		}

		beforeOK := i == 0 ||
			!isSQLIdentifierChar(
				query[i-1],
			)

		after := i + len(keyword)

		afterOK := after == len(query) ||
			!isSQLIdentifierChar(
				query[after],
			)

		if beforeOK && afterOK {
			return true
		}
	}

	return false
}

func isSQLIdentifierChar(
	c byte,
) bool {

	return unicode.IsLetter(rune(c)) ||
		unicode.IsDigit(rune(c)) ||
		c == '_' ||
		c == '$'
}

func hasMultipleStatements(query string) bool {

	query = strings.TrimSpace(query)

	// A single trailing semicolon is allowed.
	query = strings.TrimSuffix(query, ";")

	// Any remaining semicolon means multiple statements.
	return strings.Contains(query, ";")
}
