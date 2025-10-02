package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func QuerySelect(ctx context.Context, req *mcp.CallToolRequest, input QuerySelectInput) (*mcp.CallToolResult, struct{}, error) {
	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	// Build fully qualified table name
	tableName := input.Table
	if dbType == "mysql" {
		tableName = fmt.Sprintf("`%s`.`%s`", input.Database, input.Table)
	}

	// Build SELECT query using Squirrel
	query := qb.Select().From(tableName)

	// Add columns
	if len(input.Columns) > 0 {
		cols := make([]string, len(input.Columns))
		for i, col := range input.Columns {
			cols[i] = sanitizeIdentifier(col)
		}
		query = query.Columns(cols...)
	} else {
		query = query.Columns("*")
	}

	// Add WHERE conditions
	if len(input.Where) > 0 {
		query = applyWhereConditions(query, input.Where)
	}

	// Add ORDER BY
	if len(input.OrderBy) > 0 {
		for _, order := range input.OrderBy {
			query = query.OrderBy(sanitizeIdentifier(order))
		}
	}

	// Add LIMIT and OFFSET
	// Enforce max limit for SELECT queries
	limit := input.Limit
	if limit <= 0 {
		// Apply default limit if not specified
		limit = maxSelectLimit
	} else if limit > maxSelectLimit {
		// Cap at max limit if exceeded
		limit = maxSelectLimit
	}
	query = query.Limit(uint64(limit))
	
	if input.Offset > 0 {
		query = query.Offset(uint64(input.Offset))
	}

	// Execute query
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, struct{}{}, err
	}

	text := formatResults(results, fmt.Sprintf("SELECT from %s.%s", input.Database, input.Table))
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, struct{}{}, nil
}

func QueryInsert(ctx context.Context, req *mcp.CallToolRequest, input QueryInsertInput) (*mcp.CallToolResult, struct{}, error) {
	if readOnly {
		return nil, struct{}{}, fmt.Errorf("database is in read-only mode")
	}

	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	// Build fully qualified table name
	tableName := input.Table
	if dbType == "mysql" {
		tableName = fmt.Sprintf("`%s`.`%s`", input.Database, input.Table)
	}

	// Build INSERT query using Squirrel
	query := qb.Insert(tableName)

	columns := make([]string, 0, len(input.Data))
	values := make([]interface{}, 0, len(input.Data))

	for col, val := range input.Data {
		columns = append(columns, sanitizeIdentifier(col))
		values = append(values, val)
	}

	query = query.Columns(columns...).Values(values...)

	// Execute query
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build query: %w", err)
	}

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("insert failed: %w", err)
	}

	affected, _ := result.RowsAffected()
	text := fmt.Sprintf("✓ INSERT successful\n\nInserted %d row(s) into %s.%s", affected, input.Database, input.Table)
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, struct{}{}, nil
}

func QueryUpdate(ctx context.Context, req *mcp.CallToolRequest, input QueryUpdateInput) (*mcp.CallToolResult, struct{}, error) {
	if readOnly {
		return nil, struct{}{}, fmt.Errorf("database is in read-only mode")
	}

	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	if len(input.Where) == 0 {
		return nil, struct{}{}, fmt.Errorf("WHERE clause is required for UPDATE")
	}

	// Build fully qualified table name
	tableName := input.Table
	if dbType == "mysql" {
		tableName = fmt.Sprintf("`%s`.`%s`", input.Database, input.Table)
	}

	// Check row count before updating (enforce limit)
	countQuery := qb.Select("COUNT(*)").From(tableName)
	countQuery = applyWhereConditions(countQuery, input.Where)
	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build count query: %w", err)
	}

	var rowCount int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&rowCount); err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to check row count: %w", err)
	}

	if rowCount > maxUpdateLimit {
		return nil, struct{}{}, fmt.Errorf("UPDATE would affect %d row(s), which exceeds the maximum limit of %d. Please refine your WHERE clause to target fewer rows", rowCount, maxUpdateLimit)
	}

	// Build UPDATE query using Squirrel
	query := qb.Update(tableName)

	// Add SET clauses
	for col, val := range input.Data {
		query = query.Set(sanitizeIdentifier(col), val)
	}

	// Add WHERE conditions
	query = applyWhereConditionsUpdate(query, input.Where)

	// Execute query
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build query: %w", err)
	}

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("update failed: %w", err)
	}

	affected, _ := result.RowsAffected()
	text := fmt.Sprintf("✓ UPDATE successful\n\nUpdated %d row(s) in %s.%s", affected, input.Database, input.Table)
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, struct{}{}, nil
}

func QueryDelete(ctx context.Context, req *mcp.CallToolRequest, input QueryDeleteInput) (*mcp.CallToolResult, struct{}, error) {
	if readOnly {
		return nil, struct{}{}, fmt.Errorf("database is in read-only mode")
	}

	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	if len(input.Where) == 0 {
		return nil, struct{}{}, fmt.Errorf("WHERE clause is required for DELETE")
	}

	// Build fully qualified table name
	tableName := input.Table
	if dbType == "mysql" {
		tableName = fmt.Sprintf("`%s`.`%s`", input.Database, input.Table)
	}

	// Check row count before deleting (enforce limit)
	countQuery := qb.Select("COUNT(*)").From(tableName)
	countQuery = applyWhereConditions(countQuery, input.Where)
	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build count query: %w", err)
	}

	var rowCount int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&rowCount); err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to check row count: %w", err)
	}

	if rowCount > maxDeleteLimit {
		return nil, struct{}{}, fmt.Errorf("DELETE would affect %d row(s), which exceeds the maximum limit of %d. Please refine your WHERE clause to target fewer rows", rowCount, maxDeleteLimit)
	}

	// Build DELETE query using Squirrel
	query := qb.Delete(tableName)

	// Add WHERE conditions
	query = applyWhereConditionsDelete(query, input.Where)

	// Execute query
	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to build query: %w", err)
	}

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("delete failed: %w", err)
	}

	affected, _ := result.RowsAffected()
	text := fmt.Sprintf("✓ DELETE successful\n\nDeleted %d row(s) from %s.%s", affected, input.Database, input.Table)
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, struct{}{}, nil
}

func QueryRaw(ctx context.Context, req *mcp.CallToolRequest, input QueryRawInput) (*mcp.CallToolResult, struct{}, error) {
	if !allowRawQuery {
		return nil, struct{}{}, fmt.Errorf("raw SQL queries are blocked. Set ALLOW_RAW_QUERY=true to enable this dangerous feature")
	}

	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	// Switch to the specified database for MySQL
	if dbType == "mysql" {
		_, err := db.ExecContext(ctx, fmt.Sprintf("USE `%s`", input.Database))
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("failed to switch to database %s: %w", input.Database, err)
		}
	}

	isSelect := strings.HasPrefix(strings.ToUpper(strings.TrimSpace(input.Query)), "SELECT")

	if isSelect {
		rows, err := db.QueryContext(ctx, input.Query, input.Params...)
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		results, err := scanRows(rows)
		if err != nil {
			return nil, struct{}{}, err
		}

		text := formatResults(results, "Raw query successful")
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: text,
				},
			},
		}, struct{}{}, nil
	}

	if readOnly {
		return nil, struct{}{}, fmt.Errorf("database is in read-only mode")
	}

	result, err := db.ExecContext(ctx, input.Query, input.Params...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("query failed: %w", err)
	}

	affected, _ := result.RowsAffected()
	text := fmt.Sprintf("✓ Raw query successful\n\nAffected %d row(s)", affected)
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}, struct{}{}, nil
}

