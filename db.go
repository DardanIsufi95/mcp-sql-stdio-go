package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var db *sql.DB
var dbType string
var dbNames []string
var readOnly bool
var allowRawQuery bool
var qb sq.StatementBuilderType
var maxSelectLimit int
var maxUpdateLimit int
var maxDeleteLimit int

func initDatabase() error {
	dbType = getEnv("DB_TYPE", "postgres")
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbNamesStr := getEnv("DB_NAME", "postgres")
	readOnly = getEnv("DB_READONLY", "false") == "true"
	allowRawQuery = getEnv("ALLOW_RAW_QUERY", "false") == "true"
	maxSelectLimit = getEnvInt("MAX_SELECT_LIMIT", 1000)
	maxUpdateLimit = getEnvInt("MAX_UPDATE_LIMIT", 1)
	maxDeleteLimit = getEnvInt("MAX_DELETE_LIMIT", 1)
	
	// SSL configuration (default to require SSL for security)
	sslMode := getEnv("DB_SSLMODE", "require")         // PostgreSQL: disable, require, verify-ca, verify-full
	tlsMode := getEnv("DB_TLS", "true")                // MySQL: true, false, skip-verify, preferred
	sslCert := getEnv("DB_SSLCERT", "")                // Path to client certificate
	sslKey := getEnv("DB_SSLKEY", "")                  // Path to client key
	sslRootCert := getEnv("DB_SSLROOTCERT", "")        // Path to CA certificate

	// Parse comma-separated database names
	dbNames = strings.Split(dbNamesStr, ",")
	for i, name := range dbNames {
		dbNames[i] = strings.TrimSpace(name)
	}

	// Use first database for connection
	primaryDB := dbNames[0]

	var connStr string
	var err error

	if dbType == "postgres" {
		if port == "" {
			port = "5432"
		}
		
		// Build PostgreSQL connection string with SSL support
		connStr = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, primaryDB, sslMode,
		)
		
		// Add SSL certificate paths if provided
		if sslCert != "" {
			connStr += fmt.Sprintf(" sslcert=%s", sslCert)
		}
		if sslKey != "" {
			connStr += fmt.Sprintf(" sslkey=%s", sslKey)
		}
		if sslRootCert != "" {
			connStr += fmt.Sprintf(" sslrootcert=%s", sslRootCert)
		}
		
		db, err = sql.Open("postgres", connStr)
		// Use PostgreSQL placeholder format ($1, $2, etc.)
		qb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db)
	} else if dbType == "mysql" {
		if port == "" {
			port = "3306"
		}
		
		// Build MySQL connection string with TLS support
		connStr = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?parseTime=true",
			user, password, host, port, primaryDB,
		)
		
		// Add TLS configuration for MySQL
		if tlsMode == "true" {
			connStr += "&tls=true"
		} else if tlsMode == "skip-verify" {
			connStr += "&tls=skip-verify"
		} else if tlsMode == "preferred" {
			connStr += "&tls=preferred"
		} else if tlsMode == "false" {
			connStr += "&tls=false"
		}
		
		db, err = sql.Open("mysql", connStr)
		// Use MySQL placeholder format (?)
		qb = sq.StatementBuilder.PlaceholderFormat(sq.Question).RunWith(db)
	} else {
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("Connected to %s database(s): %v", dbType, dbNames)
	log.Printf("Primary database: %s", primaryDB)
	log.Printf("Read-only mode: %v", readOnly)
	log.Printf("Raw queries allowed: %v", allowRawQuery)
	log.Printf("Query limits - SELECT: %d, UPDATE: %d, DELETE: %d", maxSelectLimit, maxUpdateLimit, maxDeleteLimit)
	
	// Log SSL/TLS configuration
	if dbType == "postgres" {
		log.Printf("SSL mode: %s", sslMode)
		if sslCert != "" {
			log.Printf("SSL cert: %s", sslCert)
		}
		if sslKey != "" {
			log.Printf("SSL key: %s", sslKey)
		}
		if sslRootCert != "" {
			log.Printf("SSL root cert: %s", sslRootCert)
		}
	} else if dbType == "mysql" {
		log.Printf("TLS mode: %s", tlsMode)
	}
	
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func validateDatabase(database string) error {
	// Check if the database is in the allowed list
	for _, allowedDB := range dbNames {
		if database == allowedDB {
			return nil
		}
	}
	return fmt.Errorf("access to database '%s' not allowed (allowed: %v)", database, dbNames)
}

