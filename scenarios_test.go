package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	})
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	setupTables(context.Background(), db)
	return db, func() { db.Close() }
}

func TestRequired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	err := tm.WithTx(ctx, func(tx *sql.Tx) error {
		return SaveUser(ctx, tx, User{ID: "u1", Name: "Alice"})
	})
	if err != nil {
		t.Errorf("REQUIRED scenario failed: %v", err)
	}
}

func TestSupports(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	// Without transaction
	err := LogAccess(ctx, tm.DB, "u1")
	if err != nil {
		t.Errorf("SUPPORTS (no tx) failed: %v", err)
	}

	// With transaction
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		return LogAccess(ctx, tx, "u1")
	})
	if err != nil {
		t.Errorf("SUPPORTS (with tx) failed: %v", err)
	}
}

func TestMandatory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	// Without transaction
	err := UpdateProfileMandatoryTx(ctx, tm.DB, User{ID: "u1", Name: "NewName"})
	if err == nil {
		t.Errorf("MANDATORY (no tx) should have failed")
	}

	// With transaction
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		return UpdateProfileMandatoryTx(ctx, tx, User{ID: "u1", Name: "InsideTx"})
	})
	if err != nil {
		t.Errorf("MANDATORY (with tx) failed: %v", err)
	}
}

func TestRequiresNew(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	err := tm.SaveAuditLogRequiresNew(ctx, "u1")
	if err != nil {
		t.Errorf("REQUIRES_NEW scenario failed: %v", err)
	}
}

func TestNotSupported(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	// Without transaction
	err := FetchConfigNotTransactional(ctx, tm.DB)
	if err != nil {
		t.Errorf("NOT_SUPPORTED (no tx) failed: %v", err)
	}

	// With transaction
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		return FetchConfigNotTransactional(ctx, tx)
	})
	if err == nil {
		t.Errorf("NOT_SUPPORTED (with tx) should have failed")
	}
}

func TestNever(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	// Without transaction
	err := HealthCheckNeverTransactional(ctx, tm.DB)
	if err != nil {
		t.Errorf("NEVER (no tx) failed: %v", err)
	}

	// With transaction
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		return HealthCheckNeverTransactional(ctx, tx)
	})
	if err == nil {
		t.Errorf("NEVER (with tx) should have failed")
	}
}

func TestNested(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tm := TxManager{DB: db}
	ctx := context.Background()

	err := tm.WithTx(ctx, func(tx *sql.Tx) error {
		return ProcessRecordNested(ctx, tx, Record{ID: "r1", Data: "valid"})
	})
	if err != nil {
		t.Errorf("NESTED scenario failed: %v", err)
	}

	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		return ProcessRecordNested(ctx, tx, Record{ID: "r2", Data: "fail"})
	})
	if err == nil {
		t.Errorf("NESTED scenario should have rolled back for invalid data")
	}
}
