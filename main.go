package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Initialize database connection
	if err := initDatabase(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "mcp-go-sql-server",
			Version: "v2.0.0",
		},
		nil,
	)

	// Register query tools
	mcp.AddTool(server, &mcp.Tool{
		Name: "query_select",
		Description: `Execute a SELECT query on the database.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public",
  "table": "users",
  "columns": ["id", "name", "email"],
  "where": [
    {"column": "status", "op": "=", "value": "active"},
    {"column": "age", "op": ">=", "value": 18}
  ],
  "order_by": ["name"],
  "limit": 10
}
` + "```" + `

**Operators:** =, !=, <, <=, >, >=, LIKE, IN, BETWEEN, IS NULL, IS NOT NULL`,
	}, QuerySelect)

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_insert",
		Description: `Insert a single row into a table. Blocked in read-only mode.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "table": "users",
  "data": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}
` + "```",
	}, QueryInsert)

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_update",
		Description: `Update rows in a table. WHERE clause is required. Blocked in read-only mode.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "table": "users",
  "data": {"status": "inactive"},
  "where": [{"column": "id", "op": "=", "value": 123}]
}
` + "```",
	}, QueryUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_delete",
		Description: `Delete rows from a table. WHERE clause is required. Blocked in read-only mode.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "table": "users",
  "where": [{"column": "id", "op": "=", "value": 123}]
}
` + "```",
	}, QueryDelete)

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_raw",
		Description: `⚠️  DANGEROUS: Execute raw SQL queries. Must be explicitly enabled.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "query": "SELECT * FROM users WHERE status = ? AND age > ?",
  "params": ["active", 18]
}
` + "```",
	}, QueryRaw)

	// Register metadata tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_databases",
		Description: "List all databases",
	}, GetDatabases)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_tables",
		Description: `List tables in a database/schema.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public"
}
` + "```",
	}, GetTables)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_table_schema",
		Description: `Get detailed schema information for a table including columns, types, keys, and foreign keys.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public",
  "table": "users"
}
` + "```",
	}, GetTableSchema)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_sequences",
		Description: `Get sequence information (PostgreSQL sequences, MySQL auto_increment).

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public"
}
` + "```",
	}, GetSequences)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_custom_types",
		Description: `Get custom type definitions (PostgreSQL only). Lists ENUMs, COMPOSITEs, and other user-defined types.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public"
}
` + "```",
	}, GetCustomTypes)

	// Register function tools
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_functions",
		Description: `Get list of all functions and stored procedures in a database/schema.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public"
}
` + "```",
	}, GetFunctions)

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_function_source",
		Description: `Get the source code of a function or stored procedure.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public",
  "name": "calculate_total"
}
` + "```",
	}, GetFunctionSource)

	mcp.AddTool(server, &mcp.Tool{
		Name: "execute_function",
		Description: `Execute a function or stored procedure with parameters. Procedures blocked in read-only mode.

**Example usage:**
` + "```json" + `
{
  "database": "mydb",
  "schema": "public",
  "name": "calculate_total",
  "params": [100, 0.15]
}
` + "```",
	}, ExecuteFunction)

	log.Printf("Starting MCP SQL server with 13 tools")

	// Run the server over stdin/stdout
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}


