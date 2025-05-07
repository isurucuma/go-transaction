package main

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

func main() {
	//dbURL := os.Getenv("DATABASE_URL")
	dbURL := "postgres://admin:admin@localhost:5432/transactionDB?sslmode=disable"
	if dbURL == "" {
		log.Fatal("Please set DATABASE_URL environment variable (e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable)")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	tm := TxManager{DB: db}
	setupTables(ctx, db)

	RunAllScenarios(ctx, &tm)
}
