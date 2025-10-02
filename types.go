package main

// ===== INPUT TYPES =====

type QuerySelectInput struct {
	Database string        `json:"database" jsonschema_description:"Database name"`
	Table    string        `json:"table" jsonschema_description:"Table name"`
	Schema   string        `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Columns  []string      `json:"columns,omitempty" jsonschema_description:"Columns to select (empty for all)"`
	Where    []WhereClause `json:"where,omitempty" jsonschema_description:"WHERE conditions"`
	OrderBy  []string      `json:"order_by,omitempty" jsonschema_description:"ORDER BY columns"`
	Limit    int           `json:"limit,omitempty" jsonschema_description:"LIMIT rows"`
	Offset   int           `json:"offset,omitempty" jsonschema_description:"OFFSET rows"`
}

type WhereClause struct {
	Column string      `json:"column" jsonschema_description:"Column name"`
	Op     string      `json:"op" jsonschema_description:"Operator: =, !=, <, >, <=, >=, LIKE, IN, BETWEEN, IS NULL, IS NOT NULL"`
	Value  interface{} `json:"value,omitempty" jsonschema_description:"Value to compare"`
}

type QueryInsertInput struct {
	Database string                 `json:"database" jsonschema_description:"Database name"`
	Table    string                 `json:"table" jsonschema_description:"Table name"`
	Schema   string                 `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Data     map[string]interface{} `json:"data" jsonschema_description:"Column:value pairs to insert"`
}

type QueryUpdateInput struct {
	Database string                 `json:"database" jsonschema_description:"Database name"`
	Table    string                 `json:"table" jsonschema_description:"Table name"`
	Schema   string                 `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Data     map[string]interface{} `json:"data" jsonschema_description:"Column:value pairs to update"`
	Where    []WhereClause          `json:"where" jsonschema_description:"WHERE conditions (required)"`
}

type QueryDeleteInput struct {
	Database string        `json:"database" jsonschema_description:"Database name"`
	Table    string        `json:"table" jsonschema_description:"Table name"`
	Schema   string        `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Where    []WhereClause `json:"where" jsonschema_description:"WHERE conditions (required)"`
}

type QueryRawInput struct {
	Database string        `json:"database" jsonschema_description:"Database name"`
	Query    string        `json:"query" jsonschema_description:"Raw SQL query"`
	Params   []interface{} `json:"params,omitempty" jsonschema_description:"Query parameters"`
}

type GetTablesInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
}

type GetTableSchemaInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Table    string `json:"table" jsonschema_description:"Table name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
}

type GetSequencesInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
}

type GetCustomTypesInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
}

type GetFunctionsInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
}

type GetFunctionSourceInput struct {
	Database string `json:"database" jsonschema_description:"Database name"`
	Schema   string `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Name     string `json:"name" jsonschema_description:"Function/procedure name"`
}

type ExecuteFunctionInput struct {
	Database string        `json:"database" jsonschema_description:"Database name"`
	Schema   string        `json:"schema,omitempty" jsonschema_description:"Schema name (PostgreSQL)"`
	Name     string        `json:"name" jsonschema_description:"Function/procedure name"`
	Params   []interface{} `json:"params,omitempty" jsonschema_description:"Function parameters"`
}

// ===== OUTPUT TYPES =====

type QueryOutput struct {
	Rows     []map[string]interface{} `json:"rows,omitempty" jsonschema_description:"Query result rows"`
	Affected int64                    `json:"affected,omitempty" jsonschema_description:"Rows affected"`
	Message  string                   `json:"message,omitempty" jsonschema_description:"Result message"`
}

type ListOutput struct {
	Items []string `json:"items" jsonschema_description:"List of items"`
}

type SchemaOutput struct {
	Columns []ColumnInfo `json:"columns" jsonschema_description:"Table columns"`
}

type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default,omitempty"`
	PrimaryKey bool   `json:"primary_key,omitempty"`
}

type TextOutput struct {
	Text string `json:"text" jsonschema_description:"Text output"`
}

