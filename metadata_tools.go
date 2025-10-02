package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func GetDatabases(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, struct{}, error) {
	// Return the configured database allowlist
	var output strings.Builder
	for _, db := range dbNames {
		output.WriteString(fmt.Sprintf("• %s\n", db))
	}
	
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output.String(),
			},
		},
	}, struct{}{}, nil
}

func GetTables(ctx context.Context, req *mcp.CallToolRequest, input GetTablesInput) (*mcp.CallToolResult, struct{}, error) {
	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	var query string
	var args []interface{}

	if dbType == "postgres" {
		schema := input.Schema
		if schema == "" {
			schema = "public"
		}
		query = "SELECT tablename FROM pg_tables WHERE schemaname = $1 ORDER BY tablename"
		args = []interface{}{schema}
	} else {
		// MySQL: Use SHOW TABLES FROM database_name
		query = fmt.Sprintf("SHOW TABLES FROM `%s`", input.Database)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, struct{}{}, fmt.Errorf("failed to get tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, struct{}{}, err
		}
		tables = append(tables, table)
	}

	var output strings.Builder
	for _, table := range tables {
		output.WriteString(fmt.Sprintf("• %s\n", table))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output.String(),
			},
		},
	}, struct{}{}, nil
}

func GetTableSchema(ctx context.Context, req *mcp.CallToolRequest, input GetTableSchemaInput) (*mcp.CallToolResult, struct{}, error) {
	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	var result string
	if dbType == "postgres" {
		result = getPostgreSQLTableSchema(ctx, input)
	} else {
		result = getMySQLTableSchema(ctx, input)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, struct{}{}, nil
}

func getPostgreSQLTableSchema(ctx context.Context, input GetTableSchemaInput) string {
	schema := input.Schema
	if schema == "" {
		schema = "public"
	}

	// Get columns
	columnsQuery := `
		SELECT 
			column_name, data_type, is_nullable, column_default,
			character_maximum_length, numeric_precision, numeric_scale
		FROM information_schema.columns
		WHERE table_catalog = $1 AND table_schema = $2 AND table_name = $3
		ORDER BY ordinal_position`

	rows, err := db.QueryContext(ctx, columnsQuery, input.Database, schema, input.Table)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer rows.Close()

	output := fmt.Sprintf("Table: %s.%s.%s\n\n", input.Database, schema, input.Table)
	output += "Columns:\n"
	output += fmt.Sprintf("%-20s %-20s %-10s %-15s\n", "Column", "Type", "Nullable", "Default")
	output += fmt.Sprintf("%s\n", "-------------------------------------------------------------------")

	for rows.Next() {
		var colName, dataType, isNullable string
		var colDefault sql.NullString
		var charMaxLen, numPrecision, numScale sql.NullInt64

		rows.Scan(&colName, &dataType, &isNullable, &colDefault, &charMaxLen, &numPrecision, &numScale)

		if charMaxLen.Valid {
			dataType += fmt.Sprintf("(%d)", charMaxLen.Int64)
		} else if numPrecision.Valid {
			if numScale.Valid {
				dataType += fmt.Sprintf("(%d,%d)", numPrecision.Int64, numScale.Int64)
			} else {
				dataType += fmt.Sprintf("(%d)", numPrecision.Int64)
			}
		}

		defaultVal := ""
		if colDefault.Valid {
			defaultVal = colDefault.String
		}

		output += fmt.Sprintf("%-20s %-20s %-10s %-15s\n", colName, dataType, isNullable, defaultVal)
	}

	// Get foreign keys
	fkQuery := `
		SELECT
			kcu.column_name,
			ccu.table_schema AS foreign_schema,
			ccu.table_name AS foreign_table,
			ccu.column_name AS foreign_column
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2`

	fkRows, err := db.QueryContext(ctx, fkQuery, schema, input.Table)
	if err == nil {
		defer fkRows.Close()
		hasFKs := false
		for fkRows.Next() {
			if !hasFKs {
				output += "\nForeign Keys:\n"
				hasFKs = true
			}
			var colName, fkSchema, fkTable, fkColumn string
			fkRows.Scan(&colName, &fkSchema, &fkTable, &fkColumn)
			output += fmt.Sprintf("• %s → %s.%s(%s)\n", colName, fkSchema, fkTable, fkColumn)
		}
	}

	// Get indexes
	indexQuery := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2`

	indexRows, err := db.QueryContext(ctx, indexQuery, schema, input.Table)
	if err == nil {
		defer indexRows.Close()
		hasIndexes := false
		for indexRows.Next() {
			if !hasIndexes {
				output += "\nIndexes:\n"
				hasIndexes = true
			}
			var indexName, indexDef string
			indexRows.Scan(&indexName, &indexDef)
			output += fmt.Sprintf("• %s\n", indexName)
		}
	}

	return output
}

func getMySQLTableSchema(ctx context.Context, input GetTableSchemaInput) string {
	// Get columns - specify the database in the table reference
	columnsQuery := `
		SELECT 
			COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT,
			COLUMN_KEY, EXTRA, CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION, NUMERIC_SCALE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	rows, err := db.QueryContext(ctx, columnsQuery, input.Database, input.Table)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer rows.Close()

	output := fmt.Sprintf("Table: %s.%s\n\n", input.Database, input.Table)
	output += "Columns:\n"
	output += fmt.Sprintf("%-20s %-20s %-10s %-15s %-10s %-15s\n", "Column", "Type", "Nullable", "Default", "Key", "Extra")
	output += fmt.Sprintf("%s\n", "-----------------------------------------------------------------------------------------")

	for rows.Next() {
		var colName, dataType, isNullable, key, extra string
		var colDefault sql.NullString
		var charMaxLen, numPrecision, numScale sql.NullInt64

		rows.Scan(&colName, &dataType, &isNullable, &colDefault, &key, &extra, &charMaxLen, &numPrecision, &numScale)

		if charMaxLen.Valid {
			dataType += fmt.Sprintf("(%d)", charMaxLen.Int64)
		} else if numPrecision.Valid {
			if numScale.Valid {
				dataType += fmt.Sprintf("(%d,%d)", numPrecision.Int64, numScale.Int64)
			} else {
				dataType += fmt.Sprintf("(%d)", numPrecision.Int64)
			}
		}

		defaultVal := ""
		if colDefault.Valid {
			defaultVal = colDefault.String
		}

		output += fmt.Sprintf("%-20s %-20s %-10s %-15s %-10s %-15s\n", colName, dataType, isNullable, defaultVal, key, extra)
	}

	// Get foreign keys
	fkQuery := `
		SELECT
			COLUMN_NAME,
			REFERENCED_TABLE_SCHEMA,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL`

	fkRows, err := db.QueryContext(ctx, fkQuery, input.Database, input.Table)
	if err == nil {
		defer fkRows.Close()
		hasFKs := false
		for fkRows.Next() {
			if !hasFKs {
				output += "\nForeign Keys:\n"
				hasFKs = true
			}
			var colName, fkSchema, fkTable, fkColumn string
			fkRows.Scan(&colName, &fkSchema, &fkTable, &fkColumn)
			output += fmt.Sprintf("• %s → %s.%s(%s)\n", colName, fkSchema, fkTable, fkColumn)
		}
	}

	// Get indexes
	indexQuery := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", input.Database, input.Table)
	indexRows, err := db.QueryContext(ctx, indexQuery)
	if err == nil {
		defer indexRows.Close()
		hasIndexes := false
		indexMap := make(map[string]string)
		
		for indexRows.Next() {
			var tableName, nonUnique, keyName, seqInIndex, columnName, collation, cardinality, subPart, packed, null, indexType, comment, indexComment, visible, expression sql.NullString
			indexRows.Scan(&tableName, &nonUnique, &keyName, &seqInIndex, &columnName, &collation, &cardinality, &subPart, &packed, &null, &indexType, &comment, &indexComment, &visible, &expression)
			
			if keyName.Valid && keyName.String != "PRIMARY" {
				indexType := "INDEX"
				if nonUnique.Valid && nonUnique.String == "0" {
					indexType = "UNIQUE"
				}
				indexMap[keyName.String] = indexType
			}
		}
		
		if len(indexMap) > 0 {
			output += "\nIndexes:\n"
			for indexName, indexType := range indexMap {
				output += fmt.Sprintf("• %s (%s)\n", indexName, indexType)
			}
			hasIndexes = true
		}
		
		if !hasIndexes {
			// output += "\nNo indexes found\n"
		}
	}

	return output
}

func GetSequences(ctx context.Context, req *mcp.CallToolRequest, input GetSequencesInput) (*mcp.CallToolResult, struct{}, error) {
	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	var result string

	if dbType == "postgres" {
		schema := input.Schema
		if schema == "" {
			schema = "public"
		}

		query := `
			SELECT sequence_name, data_type, start_value, minimum_value, maximum_value, increment
			FROM information_schema.sequences
			WHERE sequence_catalog = $1 AND sequence_schema = $2
			ORDER BY sequence_name`

	rows, err := db.QueryContext(ctx, query, input.Database, schema)
	if err != nil {
		return nil, struct{}{}, err
	}
	defer rows.Close()

		result = fmt.Sprintf("Sequences in %s.%s:\n\n", input.Database, schema)
		hasSequences := false
		for rows.Next() {
			hasSequences = true
			var name, dataType, startVal, minVal, maxVal, increment string
			rows.Scan(&name, &dataType, &startVal, &minVal, &maxVal, &increment)
			result += fmt.Sprintf("• %s\n", name)
			result += fmt.Sprintf("  Type: %s\n", dataType)
			result += fmt.Sprintf("  Start: %s, Min: %s, Max: %s, Increment: %s\n\n", startVal, minVal, maxVal, increment)
		}
		if !hasSequences {
			result += "No sequences found"
		}
	} else {
		// MySQL auto_increment
		query := `
			SELECT 
				TABLE_NAME, COLUMN_NAME, DATA_TYPE, COLUMN_DEFAULT, EXTRA
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND EXTRA LIKE '%auto_increment%'
			ORDER BY TABLE_NAME, ORDINAL_POSITION`

		rows, err := db.QueryContext(ctx, query, input.Database)
		if err != nil {
			return nil, struct{}{}, err
		}
		defer rows.Close()

		result = fmt.Sprintf("Auto-increment columns in %s:\n\n", input.Database)
		hasAI := false
		for rows.Next() {
			hasAI = true
			var tableName, colName, dataType, colDefault, extra string
			rows.Scan(&tableName, &colName, &dataType, &colDefault, &extra)
			result += fmt.Sprintf("• %s.%s\n", tableName, colName)
			result += fmt.Sprintf("  Type: %s\n\n", dataType)
		}
		if !hasAI {
			result += "No auto-increment columns found"
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, struct{}{}, nil
}

func GetCustomTypes(ctx context.Context, req *mcp.CallToolRequest, input GetCustomTypesInput) (*mcp.CallToolResult, struct{}, error) {
	if dbType != "postgres" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Custom types are only supported in PostgreSQL",
				},
			},
		}, struct{}{}, nil
	}

	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	schema := input.Schema
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT 
			t.typname as type_name,
			t.typtype as type_kind,
			CASE t.typtype
				WHEN 'e' THEN 'enum'
				WHEN 'c' THEN 'composite'
				WHEN 'd' THEN 'domain'
				WHEN 'b' THEN 'base'
				ELSE 'other'
			END as type_category
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE n.nspname = $1 AND t.typtype IN ('e', 'c', 'd')
		ORDER BY t.typname`

	rows, err := db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, struct{}{}, err
	}
	defer rows.Close()

	result := fmt.Sprintf("Custom types in %s.%s:\n\n", input.Database, schema)
	hasTypes := false
	for rows.Next() {
		hasTypes = true
		var typeName, typeKind, typeCategory string
		rows.Scan(&typeName, &typeKind, &typeCategory)
		result += fmt.Sprintf("• %s (%s)\n", typeName, typeCategory)

		// Get enum values
		if typeKind == "e" {
			enumQuery := `
				SELECT enumlabel
				FROM pg_enum
				WHERE enumtypid = (SELECT oid FROM pg_type WHERE typname = $1)
				ORDER BY enumsortorder`
			enumRows, err := db.QueryContext(ctx, enumQuery, typeName)
			if err == nil {
				defer enumRows.Close()
				var values []string
				for enumRows.Next() {
					var val string
					enumRows.Scan(&val)
					values = append(values, val)
				}
				result += fmt.Sprintf("  Values: %v\n", values)
			}
		}
		result += "\n"
	}

	if !hasTypes {
		result += "No custom types found"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, struct{}{}, nil
}

