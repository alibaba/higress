package gorm

import (
	"fmt"

	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DBClient is a struct to handle PostgreSQL connections and operations
type DBClient struct {
	db *gorm.DB
}

// NewDBClient creates a new DBClient instance and establishes a connection to the PostgreSQL database
func NewDBClient(dsn string, dbType string) (*DBClient, error) {
	var db *gorm.DB
	var err error
	if dbType == "postgres" {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else if dbType == "clickhouse" {
		db, err = gorm.Open(clickhouse.Open(dsn), &gorm.Config{})
	} else if dbType == "mysql" {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	} else if dbType == "sqlite" {
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	} else {
		return nil, fmt.Errorf("unsupported database type %s", dbType)
	}
	// Connect to the database
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DBClient{db: db}, nil
}

// ExecuteSQL executes a raw SQL query and returns the result as a slice of maps
func (c *DBClient) ExecuteSQL(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := c.db.Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Prepare a slice to hold the results
	var results []map[string]interface{}

	// Iterate over the rows
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columnsData := make([]interface{}, len(columns))
		columnsPointers := make([]interface{}, len(columns))
		for i := range columnsData {
			columnsPointers[i] = &columnsData[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnsPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map to hold the column name and value
		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := columnsData[i]
			b, ok := val.([]byte)
			if ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}

		// Append the map to the results slice
		results = append(results, rowMap)
	}

	return results, nil
}
