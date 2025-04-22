package gorm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBClient is a struct to handle database connections and operations
type DBClient struct {
	db         *gorm.DB
	dsn        string
	dbType     string
	reconnect  chan struct{}
	stop       chan struct{}
	panicCount int32 // Add panic counter
}

// NewDBClient creates a new DBClient instance and establishes a connection to the database
func NewDBClient(dsn string, dbType string, stop chan struct{}) *DBClient {
	client := &DBClient{
		dsn:       dsn,
		dbType:    dbType,
		reconnect: make(chan struct{}, 1),
		stop:      stop,
	}

	// Start reconnection goroutine
	go client.reconnectLoop()

	// Try initial connection
	if err := client.connect(); err != nil {
		api.LogErrorf("Initial database connection failed: %v", err)
	}

	return client
}

func (c *DBClient) connect() error {
	var db *gorm.DB
	var err error
	gormConfig := gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch c.dbType {
	case "postgres":
		db, err = gorm.Open(postgres.Open(c.dsn), &gormConfig)
	case "clickhouse":
		db, err = gorm.Open(clickhouse.Open(c.dsn), &gormConfig)
	case "mysql":
		db, err = gorm.Open(mysql.Open(c.dsn), &gormConfig)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(c.dsn), &gormConfig)
	default:
		return fmt.Errorf("unsupported database type %s", c.dbType)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	c.db = db
	return nil
}

func (c *DBClient) reconnectLoop() {
	defer func() {
		if r := recover(); r != nil {
			api.LogErrorf("Recovered from panic in reconnectLoop: %v", r)

			// Increment panic counter
			atomic.AddInt32(&c.panicCount, 1)

			// If panic count exceeds threshold, stop trying to reconnect
			if atomic.LoadInt32(&c.panicCount) > 3 {
				api.LogErrorf("Too many panics in reconnectLoop, stopping reconnection attempts")
				return
			}

			// Wait for a while before restarting
			time.Sleep(5 * time.Second)

			// Restart the reconnect loop
			go c.reconnectLoop()
		}
	}()

	ticker := time.NewTicker(30 * time.Second) // Try to reconnect every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			api.LogInfof("Database %s connection closed", c.dbType)
			return
		case <-ticker.C:
			if c.db == nil || c.Ping() != nil {
				if err := c.connect(); err != nil {
					api.LogErrorf("Database reconnection failed: %v", err)
				} else {
					api.LogInfof("Database reconnected successfully")
					// Reset panic count on successful connection
					atomic.StoreInt32(&c.panicCount, 0)
				}
			}
		case <-c.reconnect:
			if err := c.connect(); err != nil {
				api.LogErrorf("Database reconnection failed: %v", err)
			} else {
				api.LogInfof("Database reconnected successfully")
				// Reset panic count on successful connection
				atomic.StoreInt32(&c.panicCount, 0)
			}
		}
	}
}

// ExecuteSQL executes a raw SQL query and returns the result as a slice of maps
func (c *DBClient) ExecuteSQL(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if c.db == nil {
		// Trigger reconnection
		select {
		case c.reconnect <- struct{}{}:
		default:
		}
		return nil, fmt.Errorf("database is not connected, attempting to reconnect")
	}

	rows, err := c.db.Raw(query, args...).Rows()
	if err != nil {
		// If execution fails, connection might be lost, trigger reconnection
		select {
		case c.reconnect <- struct{}{}:
		default:
		}
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

func (c *DBClient) Ping() error {
	if c.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Use context to set timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Try to ping the database
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying *sql.DB: %v", err)
	}

	return sqlDB.PingContext(ctx)
}
