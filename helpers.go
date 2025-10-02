package main

import (
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

func sanitizeIdentifier(identifier string) string {
	// Basic identifier validation - allow alphanumeric, underscore, dot, and space (for ORDER BY DESC/ASC)
	// This prevents SQL injection in identifiers
	identifier = strings.TrimSpace(identifier)
	for _, char := range identifier {
		if !(char >= 'a' && char <= 'z') &&
			!(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') &&
			char != '_' && char != ' ' && char != '.' {
			return ""
		}
	}
	return identifier
}

func applyWhereConditions(query sq.SelectBuilder, clauses []WhereClause) sq.SelectBuilder {
	for _, clause := range clauses {
		col := sanitizeIdentifier(clause.Column)
		if col == "" {
			continue
		}
		op := strings.ToUpper(clause.Op)

		switch op {
		case "=":
			query = query.Where(sq.Eq{col: clause.Value})
		case "!=", "<>":
			query = query.Where(sq.NotEq{col: clause.Value})
		case ">":
			query = query.Where(sq.Gt{col: clause.Value})
		case ">=":
			query = query.Where(sq.GtOrEq{col: clause.Value})
		case "<":
			query = query.Where(sq.Lt{col: clause.Value})
		case "<=":
			query = query.Where(sq.LtOrEq{col: clause.Value})
		case "LIKE", "ILIKE":
			query = query.Where(sq.Like{col: clause.Value})
		case "IN":
			query = query.Where(sq.Eq{col: clause.Value})
		case "NOT IN":
			query = query.Where(sq.NotEq{col: clause.Value})
		case "IS NULL":
			query = query.Where(sq.Eq{col: nil})
		case "IS NOT NULL":
			query = query.Where(sq.NotEq{col: nil})
		case "BETWEEN":
			// BETWEEN expects a slice/array with 2 values
			query = query.Where(sq.Expr(col+" BETWEEN ? AND ?", clause.Value))
		default:
			// For operators not directly supported, use Expr (still parameterized)
			query = query.Where(sq.Expr(col+" "+op+" ?", clause.Value))
		}
	}
	return query
}

func applyWhereConditionsUpdate(query sq.UpdateBuilder, clauses []WhereClause) sq.UpdateBuilder {
	for _, clause := range clauses {
		col := sanitizeIdentifier(clause.Column)
		if col == "" {
			continue
		}
		op := strings.ToUpper(clause.Op)

		switch op {
		case "=":
			query = query.Where(sq.Eq{col: clause.Value})
		case "!=", "<>":
			query = query.Where(sq.NotEq{col: clause.Value})
		case ">":
			query = query.Where(sq.Gt{col: clause.Value})
		case ">=":
			query = query.Where(sq.GtOrEq{col: clause.Value})
		case "<":
			query = query.Where(sq.Lt{col: clause.Value})
		case "<=":
			query = query.Where(sq.LtOrEq{col: clause.Value})
		case "LIKE", "ILIKE":
			query = query.Where(sq.Like{col: clause.Value})
		case "IN":
			query = query.Where(sq.Eq{col: clause.Value})
		case "NOT IN":
			query = query.Where(sq.NotEq{col: clause.Value})
		case "IS NULL":
			query = query.Where(sq.Eq{col: nil})
		case "IS NOT NULL":
			query = query.Where(sq.NotEq{col: nil})
		default:
			query = query.Where(sq.Expr(col+" "+op+" ?", clause.Value))
		}
	}
	return query
}

func applyWhereConditionsDelete(query sq.DeleteBuilder, clauses []WhereClause) sq.DeleteBuilder {
	for _, clause := range clauses {
		col := sanitizeIdentifier(clause.Column)
		if col == "" {
			continue
		}
		op := strings.ToUpper(clause.Op)

		switch op {
		case "=":
			query = query.Where(sq.Eq{col: clause.Value})
		case "!=", "<>":
			query = query.Where(sq.NotEq{col: clause.Value})
		case ">":
			query = query.Where(sq.Gt{col: clause.Value})
		case ">=":
			query = query.Where(sq.GtOrEq{col: clause.Value})
		case "<":
			query = query.Where(sq.Lt{col: clause.Value})
		case "<=":
			query = query.Where(sq.LtOrEq{col: clause.Value})
		case "LIKE", "ILIKE":
			query = query.Where(sq.Like{col: clause.Value})
		case "IN":
			query = query.Where(sq.Eq{col: clause.Value})
		case "NOT IN":
			query = query.Where(sq.NotEq{col: clause.Value})
		case "IS NULL":
			query = query.Where(sq.Eq{col: nil})
		case "IS NOT NULL":
			query = query.Where(sq.NotEq{col: nil})
		default:
			query = query.Where(sq.Expr(col+" "+op+" ?", clause.Value))
		}
	}
	return query
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

func formatResults(rows []map[string]interface{}, title string) string {
	if len(rows) == 0 {
		return fmt.Sprintf("✓ %s\n\nNo rows found", title)
	}

	// Get column names from first row
	var columns []string
	for col := range rows[0] {
		columns = append(columns, col)
	}

	// Build markdown table
	var result strings.Builder
	result.WriteString(fmt.Sprintf("✓ %s\n\nFound %d row(s):\n\n", title, len(rows)))

	// Header
	result.WriteString("|")
	for _, col := range columns {
		result.WriteString(fmt.Sprintf(" %s |", col))
	}
	result.WriteString("\n|")
	for range columns {
		result.WriteString(" --- |")
	}
	result.WriteString("\n")

	// Rows
	for _, row := range rows {
		result.WriteString("|")
		for _, col := range columns {
			val := row[col]
			if val == nil {
				result.WriteString(" NULL |")
			} else {
				// Limit cell content to 50 chars
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > 50 {
					valStr = valStr[:47] + "..."
				}
				result.WriteString(fmt.Sprintf(" %s |", valStr))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

func formatTextResult(text string) string {
	return text
}

