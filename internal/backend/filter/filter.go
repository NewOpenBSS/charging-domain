// Package filter provides generic SQL WHERE clause and ORDER BY builders for
// GraphQL list endpoints. It is the Go equivalent of Java FilterRequest.buildFilter().
//
// Each resource provides a column allowlist (map[string]string) that maps GraphQL
// camelCase field names to SQL snake_case column names. User-supplied key and
// operation strings are validated against these allowlists — no user data ever
// appears in the query text, only in positional argument slots.
package filter

import (
	"fmt"
	"strings"

	"go-ocs/internal/backend/graphql/model"
)

// WhereClause holds a parameterised SQL WHERE fragment and its positional arguments.
// SQL is empty string when no filters were specified — no WHERE keyword is emitted.
type WhereClause struct {
	SQL  string // e.g. "WHERE carrier_name ILIKE $1 AND iso = $2"
	Args []any
}

// allowedOperations is the whitelist of permitted SQL comparison operators.
// User-supplied operation strings are upper-cased before lookup.
var allowedOperations = map[string]bool{
	"=": true, "<>": true, "!=": true,
	"<": true, ">": true, "<=": true, ">=": true,
	"ILIKE": true, "LIKE": true,
	"IN": true, "NOT IN": true,
}

// BuildWhere constructs a parameterised SQL WHERE clause from a FilterRequest.
//
// allowedCols maps GraphQL field names (camelCase) to SQL column names (snake_case).
// Any key not present in allowedCols is rejected with an error, preventing SQL injection.
//
// wildcardCols are the SQL column names searched when req.Wildcard is set.
// They are ORed together inside parentheses and ANDed with the explicit predicates.
func BuildWhere(
	req *model.FilterRequest,
	allowedCols map[string]string,
	wildcardCols []string,
) (WhereClause, error) {
	if req == nil {
		return WhereClause{}, nil
	}

	var clauses []string
	var args []any
	idx := 1

	for _, f := range req.Filters {
		col, ok := allowedCols[f.Key]
		if !ok {
			return WhereClause{}, fmt.Errorf("filter key %q is not permitted", f.Key)
		}

		op := strings.ToUpper(strings.TrimSpace(f.Operation))
		if !allowedOperations[op] {
			return WhereClause{}, fmt.Errorf("filter operation %q is not permitted", f.Operation)
		}

		if op == "IN" || op == "NOT IN" {
			// Split comma-separated value into individual positional args
			parts := strings.Split(f.Value, ",")
			placeholders := make([]string, 0, len(parts))
			for _, p := range parts {
				args = append(args, strings.TrimSpace(p))
				placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
				idx++
			}
			clauses = append(clauses,
				fmt.Sprintf("%s %s (%s)", col, op, strings.Join(placeholders, ", ")))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s %s $%d", col, f.Operation, idx))
			args = append(args, f.Value)
			idx++
		}
	}

	// Wildcard: searched across all wildcard columns using ILIKE, ORed together
	if req.Wildcard != nil && strings.TrimSpace(*req.Wildcard) != "" {
		wcClauses := make([]string, 0, len(wildcardCols))
		for _, col := range wildcardCols {
			wcClauses = append(wcClauses, fmt.Sprintf("%s ILIKE $%d", col, idx))
		}
		if len(wcClauses) > 0 {
			clauses = append(clauses, "("+strings.Join(wcClauses, " OR ")+")")
			args = append(args, "%"+strings.TrimSpace(*req.Wildcard)+"%")
		}
	}

	if len(clauses) == 0 {
		return WhereClause{}, nil
	}
	return WhereClause{
		SQL:  "WHERE " + strings.Join(clauses, " AND "),
		Args: args,
	}, nil
}

// BuildOrderBy constructs a SQL ORDER BY clause from the sort fields in a PageRequest.
//
// defaultCol is used when no sort key is specified or the request is nil.
// allowedCols is the same per-resource map used by BuildWhere.
func BuildOrderBy(
	req *model.PageRequest,
	defaultCol string,
	allowedCols map[string]string,
) (string, error) {
	col := defaultCol
	dir := "ASC"

	if req != nil && req.Sort != nil && strings.TrimSpace(req.Sort.Key) != "" {
		mapped, ok := allowedCols[req.Sort.Key]
		if !ok {
			return "", fmt.Errorf("sort key %q is not permitted", req.Sort.Key)
		}
		col = mapped
		if strings.EqualFold(strings.TrimSpace(req.Sort.Order), "desc") {
			dir = "DESC"
		}
	}

	return fmt.Sprintf("ORDER BY %s %s", col, dir), nil
}

// PageOffset extracts LIMIT and OFFSET from a PageRequest, applying safe defaults:
// pageIndex defaults to 0, pageSize defaults to 10.
func PageOffset(req *model.PageRequest) (limit, offset int) {
	pageIndex := 0
	pageSize := 10

	if req != nil {
		if req.PageIndex != nil && *req.PageIndex > 0 {
			pageIndex = *req.PageIndex
		}
		if req.PageSize != nil && *req.PageSize > 0 {
			pageSize = *req.PageSize
		}
	}
	return pageSize, pageIndex * pageSize
}
