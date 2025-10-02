# MCP Go SQL Server

A Model Context Protocol (MCP) server for PostgreSQL and MySQL databases, implemented in Go. This is a **stdio-based** version of the [TypeScript HTTP SQL MCP server](https://github.com/DardanIsufi95/mcp-sql-http-ts).

## Features

✅ **Database Support**: PostgreSQL and MySQL  
✅ **Secure Query Builder**: Uses Squirrel query builder (like Knex for Go)  
✅ **SQL Injection Protection**: All queries use parameterized statements  
✅ **Identifier Sanitization**: Column/table names validated before use  
✅ **Secure Query Tools**: SELECT, INSERT, UPDATE, DELETE  
✅ **Raw SQL**: Execute custom queries (use with caution)  
✅ **Metadata Tools**: List databases, tables, and schemas  
✅ **Read-Only Mode**: Prevent write operations  
✅ **Connection Validation**: Database allowlist protection  
✅ **Stdio Transport**: Works with Cursor, Claude Desktop, and other MCP clients  

## Quick Start

### 1. Set Environment Variables

```bash
export DB_TYPE=postgres                          # or mysql
export DB_HOST=localhost
export DB_PORT=5432                             # or 3306 for MySQL
export DB_USER=postgres
export DB_PASSWORD=yourpassword
export DB_NAME=yourdatabase                      # or comma-separated: "db1,db2,db3"
export DB_READONLY=false                         # optional
export ALLOW_RAW_QUERY=false                     # optional
export MAX_SELECT_LIMIT=1000                     # optional
export MAX_UPDATE_LIMIT=1                        # optional
export MAX_DELETE_LIMIT=1                        # optional
```

### 2. Build and Run

```bash
go mod tidy
go build -o mcp-server.exe .
./mcp-server.exe
```

## Configuration

All configuration is done via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_TYPE` | No | `postgres` | Database type: `postgres` or `mysql` |
| `DB_HOST` | No | `localhost` | Database host |
| `DB_PORT` | No | `5432` | Database port (5432 for PostgreSQL, 3306 for MySQL) |
| `DB_USER` | No | `postgres` | Database user |
| `DB_PASSWORD` | No | `` | Database password |
| `DB_NAME` | No | `postgres` | Database name(s) to connect to (comma-separated for multiple: `"db1,db2,db3"`) |
| `DB_READONLY` | No | `false` | Enable read-only mode (`true` or `false`) |
| `ALLOW_RAW_QUERY` | No | `false` | Enable raw SQL queries ⚠️ DANGEROUS (`true` or `false`) |
| `MAX_SELECT_LIMIT` | No | `1000` | Maximum number of rows returned by SELECT queries |
| `MAX_UPDATE_LIMIT` | No | `1` | Maximum number of rows that can be updated in a single UPDATE query |
| `MAX_DELETE_LIMIT` | No | `1` | Maximum number of rows that can be deleted in a single DELETE query |

## MCP Client Configuration

### Cursor / VS Code

Create `.cursor/mcp.json` or `.vscode/mcp.json`:

```json
{
  "mcpServers": {
    "go-mcp-sql-server": {
      "command": "C:\\Users\\PC\\Desktop\\mcp-go-sql\\mcp-server.exe",
      "env": {
        "DB_TYPE": "postgres",
        "DB_HOST": "localhost",
        "DB_PORT": "5432",
        "DB_USER": "postgres",
        "DB_PASSWORD": "yourpassword",
        "DB_NAME": "yourdatabase",
        "DB_READONLY": "false",
        "ALLOW_RAW_QUERY": "false",
        "MAX_SELECT_LIMIT": "1000",
        "MAX_UPDATE_LIMIT": "1",
        "MAX_DELETE_LIMIT": "1"
      }
    }
  }
}
```

### Claude Desktop

Add to Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`  
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "go-mcp-sql-server": {
      "command": "/path/to/mcp-server",
      "env": {
        "DB_TYPE": "mysql",
        "DB_HOST": "localhost",
        "DB_PORT": "3306",
        "DB_USER": "root",
        "DB_PASSWORD": "password",
        "DB_NAME": "myapp",
        "DB_READONLY": "false",
        "ALLOW_RAW_QUERY": "false",
        "MAX_SELECT_LIMIT": "1000",
        "MAX_UPDATE_LIMIT": "1",
        "MAX_DELETE_LIMIT": "1"
      }
    }
  }
}
```

## Multiple Database Support

You can configure access to multiple databases by providing a comma-separated list:

```json
{
  "mcpServers": {
    "go-mcp-sql-server": {
      "command": "C:\\Users\\PC\\Desktop\\mcp-go-sql\\mcp-server.exe",
      "env": {
        "DB_TYPE": "mysql",
        "DB_HOST": "localhost",
        "DB_PORT": "3306",
        "DB_USER": "root",
        "DB_PASSWORD": "",
        "DB_NAME": "information_schema,myapp,testdb",
        "DB_READONLY": "false",
        "MAX_SELECT_LIMIT": "1000",
        "MAX_UPDATE_LIMIT": "1",
        "MAX_DELETE_LIMIT": "1"
      }
    }
  }
}
```

**How it works:**
- The server connects to the **first database** in the list (primary connection)
- Tools can access **any database** in the list
- The `get_databases` tool returns only the configured databases (security feature)
- Database names are validated against the allowlist on every query

**Example:**
```bash
# Configure three databases
export DB_NAME="information_schema,gzk,portals"

# Now you can query any of these:
# - information_schema.TABLES
# - gzk.users
# - portals.content
```

## Available Tools (13 Total)

The server implements **all tools** from the TypeScript version, organized into three categories:

### Query Tools (5 tools)

#### 1. `query_select` - SELECT Query

Execute SELECT queries with WHERE, ORDER BY, LIMIT, and OFFSET support.

**Input:**
```json
{
  "database": "yourdatabase",
  "table": "users",
  "columns": ["id", "name", "email"],
  "where": [
    {"column": "status", "op": "=", "value": "active"},
    {"column": "age", "op": ">", "value": 18}
  ],
  "order_by": ["name"],
  "limit": 10,
  "offset": 0
}
```

**Output:**
```
✓ SELECT from mydb.users

Found 2 row(s):

| id | name | email |
| --- | --- | --- |
| 1 | John | john@example.com |
| 2 | Jane | jane@example.com |
```

#### 2. `query_insert` - INSERT Row

Insert a single row into a table.

**Input:**
```json
{
  "database": "yourdatabase",
  "table": "users",
  "data": {
    "name": "Bob",
    "email": "bob@example.com",
    "age": 30
  }
}
```

**Output:**
```
✓ INSERT successful

Inserted 1 row(s) into yourdatabase.users
```

#### 3. `query_update` - UPDATE Rows

Update rows in a table. **WHERE clause is required** for safety.

**Input:**
```json
{
  "database": "yourdatabase",
  "table": "users",
  "data": {
    "status": "inactive"
  },
  "where": [
    {"column": "id", "op": "=", "value": 123}
  ]
}
```

**Output:**
```
✓ UPDATE successful

Updated 1 row(s) in yourdatabase.users
```

#### 4. `query_delete` - DELETE Rows

Delete rows from a table. **WHERE clause is required** for safety.

**Input:**
```json
{
  "database": "yourdatabase",
  "table": "users",
  "where": [
    {"column": "id", "op": "=", "value": 123}
  ]
}
```

**Output:**
```
✓ DELETE successful

Deleted 1 row(s) from yourdatabase.users
```

#### 5. `query_raw` - Raw SQL Query

Execute raw SQL queries. **Use with caution!**

**Input:**
```json
{
  "database": "yourdatabase",
  "query": "SELECT * FROM users WHERE status = ? AND age > ?",
  "params": ["active", 18]
}
```

**Output:**
```
✓ Raw query successful

Found 5 row(s):

| id | name | status |
| --- | --- | --- |
| 1 | Alice | active |
| 2 | Bob | active |
...
```

### Metadata Tools (5 tools)

#### 6. `get_databases` - List Databases

List databases from the configured allowlist (from `DB_NAME` environment variable).

**Input:** None

**Output:**
```
• information_schema
• myapp
• testdb
```

**Note:** This returns only the databases you've configured in `DB_NAME`, not all databases on the server. This provides security by restricting access.

#### 7. `get_tables` - List Tables

List tables in a specific database.

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public"
}
```

**Output:**
```
• users
• orders
• products
```

#### 8. `get_table_schema` - Get Table Schema

Get detailed schema information for a table, including foreign keys.

**Input:**
```json
{
  "database": "yourdatabase",
  "table": "users",
  "schema": "public"
}
```

**Output:**
```
Table: yourdatabase.users

Columns:
Column               Type                 Nullable   Default         Key        Extra          
-----------------------------------------------------------------------------------------
id                   int(11)              NO                         PRI        auto_increment 
name                 varchar(255)         NO                                                   
email                varchar(255)         YES        NULL                                      
user_id              int(11)              YES        NULL            MUL                       

Foreign Keys:
• user_id → yourdatabase.profiles(id)

Indexes:
• idx_email (UNIQUE)
• idx_name (INDEX)
```

#### 9. `get_sequences` - List Sequences

Get sequence information (PostgreSQL sequences or MySQL auto_increment columns).

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public"
}
```

**Output:**
```
Sequences in yourdatabase.public:

• users_id_seq
  Type: bigint
  Start: 1, Min: 1, Max: 9223372036854775807, Increment: 1
```

#### 10. `get_custom_types` - List Custom Types

List custom types (PostgreSQL only: ENUMs, COMPOSITEs, DOMAINs).

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public"
}
```

**Output:**
```
Custom types in yourdatabase.public:

• status_enum (enum)
  Values: [active, inactive, pending]

• address_type (composite)
  Attributes:
    - street: text
    - city: varchar(100)
```

### Function Tools (3 tools)

#### 11. `get_functions` - List Functions/Procedures

List all functions and stored procedures.

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public"
}
```

**Output:**
```
Functions and procedures in yourdatabase.public:

Functions (2):
• calculate_total(price numeric, tax_rate numeric)
  Returns: numeric | Language: plpgsql
• get_user_count()
  Returns: bigint | Language: sql

Procedures (1):
• update_user_status(user_id integer, new_status text)
  Language: plpgsql
```

#### 12. `get_function_source` - View Function Source

Get the complete source code of a function or procedure.

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public",
  "name": "calculate_total"
}
```

**Output:**
```
FUNCTION: public.calculate_total

CREATE OR REPLACE FUNCTION public.calculate_total(price numeric, tax_rate numeric)
 RETURNS numeric
 LANGUAGE plpgsql
AS $function$
BEGIN
    RETURN price * (1 + tax_rate);
END;
$function$
```

#### 13. `execute_function` - Execute Function/Procedure

Execute a function or stored procedure with parameters.

**Input:**
```json
{
  "database": "yourdatabase",
  "schema": "public",
  "name": "calculate_total",
  "params": [100, 0.15]
}
```

**Output:**
```
✓ Function executed successfully

Result: 115.00
```

**Note:** Stored procedures are blocked in read-only mode.

## Supported WHERE Operators

- `=` - Equal
- `!=` or `<>` - Not equal
- `<` - Less than
- `<=` - Less than or equal
- `>` - Greater than
- `>=` - Greater than or equal
- `LIKE` - Pattern matching
- `IN` - In list
- `BETWEEN` - Between two values
- `IS NULL` - Is null
- `IS NOT NULL` - Is not null

## Query Limits

The server enforces configurable limits on query operations to prevent accidental large-scale operations:

### SELECT Queries
- **Default limit**: 1000 rows
- **Behavior**: If no LIMIT is specified, the default is applied automatically
- **Override**: User-specified limits are capped at the maximum
- **Example**: If `MAX_SELECT_LIMIT=100`, a query with `LIMIT=200` will return max 100 rows

### UPDATE Queries
- **Default limit**: 1 row
- **Behavior**: Before executing, counts rows matching WHERE clause
- **Prevention**: If count exceeds limit, returns an error with the count
- **Error message**: "UPDATE would affect X row(s), which exceeds the maximum limit of Y"

### DELETE Queries
- **Default limit**: 1 row
- **Behavior**: Before executing, counts rows matching WHERE clause
- **Prevention**: If count exceeds limit, returns an error with the count
- **Error message**: "DELETE would affect X row(s), which exceeds the maximum limit of Y"

### INSERT Queries
- **Design**: Only accepts a single row (map of column:value pairs)
- **No explicit limit needed**

**Why these limits?**
- Prevents accidental mass deletions/updates
- Protects against poorly-formed WHERE clauses
- Forces deliberate operations for bulk changes
- Can be adjusted per environment (dev vs production)

## Security Features

✅ **Query Builder**: Uses [Squirrel](https://github.com/Masterminds/squirrel) query builder (Go equivalent of Knex.js)  
✅ **Parameterized Queries**: All values automatically escaped and parameterized  
✅ **Identifier Validation**: Column and table names sanitized before use  
✅ **No String Concatenation**: SQL built safely through query builder API  
✅ **Required WHERE clauses**: UPDATE and DELETE operations require WHERE conditions  
✅ **Query limits**: Configurable limits for SELECT, UPDATE, and DELETE operations  
✅ **Database validation**: Only configured database can be accessed  
✅ **Read-only mode**: Optionally prevent all write operations  
✅ **Connection pooling**: Managed by database/sql package  

### SQL Injection Protection

Unlike the original implementation, this version uses **Squirrel**, a mature SQL query builder that:

- Automatically handles parameter binding ($1, $2 for PostgreSQL, ? for MySQL)
- Separates SQL structure from data values
- Validates and sanitizes identifiers
- Prevents common SQL injection vectors
- Similar security model to Knex.js from the TypeScript version  

## Examples

### Example 1: Query Users

```json
{
  "tool": "query_select",
  "input": {
    "database": "myapp",
    "table": "users",
    "columns": ["id", "name", "email"],
    "where": [
      {"column": "created_at", "op": ">", "value": "2024-01-01"}
    ],
    "order_by": ["created_at DESC"],
    "limit": 5
  }
}
```

### Example 2: Insert Order

```json
{
  "tool": "query_insert",
  "input": {
    "database": "myapp",
    "table": "orders",
    "data": {
      "user_id": 123,
      "product_id": 456,
      "quantity": 2,
      "total": 99.99
    }
  }
}
```

### Example 3: Get Table Structure

```json
{
  "tool": "get_table_schema",
  "input": {
    "database": "myapp",
    "table": "products"
  }
}
```

## Development

### Project Structure

```
mcp-go-sql/
├── main.go              # Server setup and tool registration
├── types.go             # Input/output type definitions
├── db.go                # Database connection management
├── helpers.go           # Helper functions (sanitization, query building)
├── query_tools.go       # Query tools (SELECT, INSERT, UPDATE, DELETE, RAW)
├── metadata_tools.go    # Metadata tools (databases, tables, schemas, etc.)
├── function_tools.go    # Function/procedure tools
├── go.mod               # Go dependencies
├── go.sum               # Dependency checksums
├── README.md            # This file
```

### Building

```bash
# Build for current platform
go build -o mcp-server .

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o mcp-server-linux .

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o mcp-server-macos .

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o mcp-server.exe .
```

### Testing

```bash
# Test PostgreSQL connection
DB_TYPE=postgres DB_HOST=localhost DB_PORT=5432 \
DB_USER=postgres DB_PASSWORD=pass DB_NAME=testdb \
./mcp-server.exe

# Test MySQL connection
DB_TYPE=mysql DB_HOST=localhost DB_PORT=3306 \
DB_USER=root DB_PASSWORD=pass DB_NAME=testdb \
./mcp-server.exe
```

## Requirements

- Go 1.23.0 or higher
- PostgreSQL or MySQL database
- [MCP Go SDK v1.0.0](https://github.com/modelcontextprotocol/go-sdk)
- [Squirrel v1.5.4](https://github.com/Masterminds/squirrel) - SQL query builder

## Differences from TypeScript Version

This is a **stdio-based** implementation compared to the original HTTP-based TypeScript version:

| Feature | TypeScript (HTTP) | Go (stdio) |
|---------|-------------------|------------|
| Transport | HTTP with headers | stdin/stdout |
| Configuration | HTTP headers | Environment variables |
| Session Management | HTTP sessions | Single connection |
| Multi-database | Per-session (multiple) | Multiple databases per instance (comma-separated) |
| Query Builder | Knex.js | Squirrel |
| Functions/Procedures | ✅ Supported | ✅ Supported |
| Custom Types | ✅ Supported | ✅ Supported |
| Sequences | ✅ Supported | ✅ Supported |
| Tool Count | 13 tools | 13 tools |

## Feature Complete ✅

This Go implementation now has **feature parity** with the TypeScript version:

- ✅ All 13 tools implemented
- ✅ Stored procedures/functions support
- ✅ Custom types enumeration (PostgreSQL)
- ✅ Sequence listing
- ✅ Foreign key relationships
- ✅ Squirrel query builder
- ✅ Read-only mode
- ✅ Raw SQL support (opt-in)

## Future Enhancements

Potential additions beyond the TypeScript version:

- [ ] Transaction support
- [ ] Batch operations
- [ ] Multiple database connections
- [ ] Connection pooling configuration
- [ ] SSL/TLS support
- [ ] Query timeout configuration
- [ ] Query result caching
- [ ] HTTP transport option

## License

MIT

## References

- [Original TypeScript Version](https://github.com/DardanIsufi95/mcp-sql-http-ts)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [PostgreSQL Driver](https://github.com/lib/pq)
- [MySQL Driver](https://github.com/go-sql-driver/mysql)
