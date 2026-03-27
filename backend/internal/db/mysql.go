package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// InitDB initializes the connection to mgt_db
func InitDB() {
	// Root (no password), Port: 3306, DB: mgt_db
	dsn := "root:@tcp(localhost:3306)/mgt_db?parseTime=true"
	
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
		log.Printf("Warning: Could not connect to MySQL on port 3306. Is the database running? Error: %v", err)
	} else {
		fmt.Println("Successfully connected to MySQL (mgt_db) on port 3306")
	}
}
