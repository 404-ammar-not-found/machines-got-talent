package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// InitDB initializes the connection to mgt_db
func InitDB() {
	host := envOrDefault("MGT_DB_HOST", "localhost")
	port := envOrDefault("MGT_DB_PORT", "3306")
	user := envOrDefault("MGT_DB_USER", "root")
	password := os.Getenv("MGT_DB_PASSWORD")
	name := envOrDefault("MGT_DB_NAME", "mgt_db")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, name)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Check connection
	if err := DB.Ping(); err != nil {
		log.Printf("Warning: Could not connect to MySQL at %s:%s/%s. Is the database running? Error: %v", host, port, name, err)
	} else {
		fmt.Printf("Successfully connected to MySQL (%s) at %s:%s\n", name, host, port)
	}
}
