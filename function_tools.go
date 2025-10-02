package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func GetFunctions(ctx context.Context, req *mcp.CallToolRequest, input GetFunctionsInput) (*mcp.CallToolResult, struct{}, error) {
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
			SELECT 
				p.proname as name,
				CASE p.prokind
					WHEN 'f' THEN 'function'
					WHEN 'p' THEN 'procedure'
					WHEN 'a' THEN 'aggregate'
					WHEN 'w' THEN 'window'
				END as type,
				pg_catalog.pg_get_function_arguments(p.oid) as arguments,
				pg_catalog.pg_get_function_result(p.oid) as return_type,
				l.lanname as language
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			JOIN pg_language l ON l.oid = p.prolang
			WHERE n.nspname = $1
			ORDER BY p.proname`

		rows, err := db.QueryContext(ctx, query, schema)
		if err != nil {
			return nil, struct{}{}, err
		}
		defer rows.Close()

		type funcInfo struct {
			name       string
			typ        string
			arguments  string
			returnType string
			language   string
		}

		var functions []funcInfo
		for rows.Next() {
			var f funcInfo
			rows.Scan(&f.name, &f.typ, &f.arguments, &f.returnType, &f.language)
			functions = append(functions, f)
		}

		result = fmt.Sprintf("Functions and procedures in %s.%s:\n\n", input.Database, schema)

		if len(functions) == 0 {
			result += "No functions or procedures found"
		} else {
			// Group by type
			funcs := []funcInfo{}
			procs := []funcInfo{}
			others := []funcInfo{}

			for _, f := range functions {
				switch f.typ {
				case "function":
					funcs = append(funcs, f)
				case "procedure":
					procs = append(procs, f)
				default:
					others = append(others, f)
				}
			}

			if len(funcs) > 0 {
				result += fmt.Sprintf("Functions (%d):\n", len(funcs))
				for _, f := range funcs {
					result += fmt.Sprintf("• %s(%s)\n", f.name, f.arguments)
					result += fmt.Sprintf("  Returns: %s | Language: %s\n", f.returnType, f.language)
				}
				result += "\n"
			}

			if len(procs) > 0 {
				result += fmt.Sprintf("Procedures (%d):\n", len(procs))
				for _, p := range procs {
					result += fmt.Sprintf("• %s(%s)\n", p.name, p.arguments)
					result += fmt.Sprintf("  Language: %s\n", p.language)
				}
				result += "\n"
			}

			if len(others) > 0 {
				result += fmt.Sprintf("Other (%d):\n", len(others))
				for _, o := range others {
					result += fmt.Sprintf("• %s (%s)\n", o.name, o.typ)
				}
			}
		}
	} else {
		// MySQL
		query := `
			SELECT 
				ROUTINE_NAME as name,
				ROUTINE_TYPE as type,
				DTD_IDENTIFIER as return_type,
				CREATED,
				LAST_ALTERED
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_SCHEMA = ?
			ORDER BY ROUTINE_NAME`

		rows, err := db.QueryContext(ctx, query, input.Database)
		if err != nil {
			return nil, struct{}{}, err
		}
		defer rows.Close()

		type routineInfo struct {
			name       string
			typ        string
			returnType string
			created    string
			altered    string
		}

		var routines []routineInfo
		for rows.Next() {
			var r routineInfo
			rows.Scan(&r.name, &r.typ, &r.returnType, &r.created, &r.altered)
			routines = append(routines, r)
		}

		result = fmt.Sprintf("Functions and procedures in %s:\n\n", input.Database)

		if len(routines) == 0 {
			result += "No functions or procedures found"
		} else {
			functions := []routineInfo{}
			procedures := []routineInfo{}

			for _, r := range routines {
				if r.typ == "FUNCTION" {
					functions = append(functions, r)
				} else {
					procedures = append(procedures, r)
				}
			}

			if len(functions) > 0 {
				result += fmt.Sprintf("Functions (%d):\n", len(functions))
				for _, f := range functions {
					result += fmt.Sprintf("• %s\n", f.name)
					result += fmt.Sprintf("  Returns: %s\n", f.returnType)
					result += fmt.Sprintf("  Created: %s\n", f.created)
				}
				result += "\n"
			}

			if len(procedures) > 0 {
				result += fmt.Sprintf("Procedures (%d):\n", len(procedures))
				for _, p := range procedures {
					result += fmt.Sprintf("• %s\n", p.name)
					result += fmt.Sprintf("  Created: %s\n", p.created)
				}
			}
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

func GetFunctionSource(ctx context.Context, req *mcp.CallToolRequest, input GetFunctionSourceInput) (*mcp.CallToolResult, struct{}, error) {
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
			SELECT 
				p.proname as name,
				CASE p.prokind
					WHEN 'f' THEN 'function'
					WHEN 'p' THEN 'procedure'
				END as type,
				pg_catalog.pg_get_functiondef(p.oid) as definition
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE n.nspname = $1 AND p.proname = $2`

		rows, err := db.QueryContext(ctx, query, schema, input.Name)
		if err != nil {
			return nil, struct{}{}, err
		}
		defer rows.Close()

		if !rows.Next() {
			result = fmt.Sprintf("Function or procedure '%s' not found in %s", input.Name, schema)
		} else {
			var name, typ, definition string
			rows.Scan(&name, &typ, &definition)
			result = fmt.Sprintf("%s: %s.%s\n\n%s", strings.ToUpper(typ), schema, name, definition)
		}
	} else {
		// MySQL
		query := `
			SELECT 
				ROUTINE_NAME as name,
				ROUTINE_TYPE as type,
				ROUTINE_DEFINITION as definition,
				ROUTINE_SCHEMA
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ?`

		rows, err := db.QueryContext(ctx, query, input.Database, input.Name)
		if err != nil {
			return nil, struct{}{}, err
		}
		defer rows.Close()

		if !rows.Next() {
			result = fmt.Sprintf("Function or procedure '%s' not found in %s", input.Name, input.Database)
		} else {
			var name, typ, definition, schema string
			rows.Scan(&name, &typ, &definition, &schema)
			if definition == "" {
				definition = "Source code not available"
			}
			result = fmt.Sprintf("%s: %s.%s\n\n%s", typ, schema, name, definition)
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

func ExecuteFunction(ctx context.Context, req *mcp.CallToolRequest, input ExecuteFunctionInput) (*mcp.CallToolResult, struct{}, error) {
	if err := validateDatabase(input.Database); err != nil {
		return nil, struct{}{}, err
	}

	var result string

	if dbType == "postgres" {
		schema := input.Schema
		if schema == "" {
			schema = "public"
		}

		// Determine if it's a function or procedure
		typeQuery := `
			SELECT prokind FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE n.nspname = $1 AND p.proname = $2`

		var prokind string
		err := db.QueryRowContext(ctx, typeQuery, schema, input.Name).Scan(&prokind)
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("function or procedure '%s' not found in %s", input.Name, schema)
		}

		isProcedure := prokind == "p"

		// Check read-only for procedures
		if isProcedure && readOnly {
			return nil, struct{}{}, fmt.Errorf("stored procedures are not allowed in read-only mode")
		}

		// Build parameter placeholders
		placeholders := make([]string, len(input.Params))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		paramStr := strings.Join(placeholders, ", ")
		qualifiedName := fmt.Sprintf("%s.%s", schema, input.Name)

		if isProcedure {
			// Call procedure
			query := fmt.Sprintf("CALL %s(%s)", qualifiedName, paramStr)
			rows, err := db.QueryContext(ctx, query, input.Params...)
			if err != nil {
				return nil, struct{}{}, fmt.Errorf("procedure execution failed: %w", err)
			}
			defer rows.Close()

			results, _ := scanRows(rows)
			result = fmt.Sprintf("✓ Procedure executed successfully\n\n%v", results)
		} else {
			// Call function
			query := fmt.Sprintf("SELECT %s(%s) as result", qualifiedName, paramStr)
			var funcResult interface{}
			err := db.QueryRowContext(ctx, query, input.Params...).Scan(&funcResult)
			if err != nil {
				return nil, struct{}{}, fmt.Errorf("function execution failed: %w", err)
			}
			result = fmt.Sprintf("✓ Function executed successfully\n\nResult: %v", funcResult)
		}
	} else {
		// MySQL - determine if it's a function or procedure
		typeQuery := `
			SELECT ROUTINE_TYPE as type
			FROM INFORMATION_SCHEMA.ROUTINES
			WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ?`

		var routineType string
		err := db.QueryRowContext(ctx, typeQuery, input.Database, input.Name).Scan(&routineType)
		if err != nil {
			return nil, struct{}{}, fmt.Errorf("function or procedure '%s' not found in %s", input.Name, input.Database)
		}

		isProcedure := routineType == "PROCEDURE"

		// Check read-only for procedures
		if isProcedure && readOnly {
			return nil, struct{}{}, fmt.Errorf("stored procedures are not allowed in read-only mode")
		}

		// Switch database
		_, err = db.ExecContext(ctx, fmt.Sprintf("USE `%s`", input.Database))
		if err != nil {
			return nil, struct{}{}, err
		}

		// Build parameter placeholders
		placeholders := make([]string, len(input.Params))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		paramStr := strings.Join(placeholders, ", ")

		if isProcedure {
			// Call procedure
			query := fmt.Sprintf("CALL %s(%s)", input.Name, paramStr)
			rows, err := db.QueryContext(ctx, query, input.Params...)
			if err != nil {
				return nil, struct{}{}, fmt.Errorf("procedure execution failed: %w", err)
			}
			defer rows.Close()

			results, _ := scanRows(rows)
			result = fmt.Sprintf("✓ Procedure executed successfully\n\n%v", results)
		} else {
			// Call function
			query := fmt.Sprintf("SELECT %s(%s) as result", input.Name, paramStr)
			var funcResult interface{}
			err := db.QueryRowContext(ctx, query, input.Params...).Scan(&funcResult)
			if err != nil {
				return nil, struct{}{}, fmt.Errorf("function execution failed: %w", err)
			}
			result = fmt.Sprintf("✓ Function executed successfully\n\nResult: %v", funcResult)
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

