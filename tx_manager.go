package main

import (
	"context"
	"database/sql"
	"errors"
)

type User struct {
	ID, Name string
}

type Record struct {
	ID, Data string
}

type DBExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type TxManager struct {
	DB *sql.DB
}

func (tm *TxManager) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := tm.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// REQUIRED
func SaveUser(ctx context.Context, exec DBExecutor, user User) error {
	_, err := exec.ExecContext(ctx, "INSERT INTO users (id, name) VALUES ($1, $2)", user.ID, user.Name)
	return err
}

// SUPPORTS
func LogAccess(ctx context.Context, exec DBExecutor, userID string) error {
	_, err := exec.ExecContext(ctx, "INSERT INTO audit_log (user_id, event) VALUES ($1, 'accessed')", userID)
	return err
}

// MANDATORY
func UpdateProfileMandatoryTx(ctx context.Context, exec DBExecutor, user User) error {
	if _, ok := exec.(*sql.Tx); !ok {
		return errors.New("MANDATORY: must be in transaction")
	}
	_, err := exec.ExecContext(ctx, "UPDATE users SET name = $1 WHERE id = $2", user.Name, user.ID)
	return err
}

// REQUIRES_NEW
func (tm *TxManager) SaveAuditLogRequiresNew(ctx context.Context, userID string) error {
	return tm.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO audit_log (user_id, event) VALUES ($1, 'REGISTERED')", userID)
		return err
	})
}

// NOT_SUPPORTED
func FetchConfigNotTransactional(ctx context.Context, exec DBExecutor) error {
	if _, ok := exec.(*sql.Tx); ok {
		return errors.New("NOT_SUPPORTED: transaction not allowed")
	}
	_, err := exec.QueryContext(ctx, "SELECT key, value FROM config")
	return err
}

// NEVER
func HealthCheckNeverTransactional(ctx context.Context, exec DBExecutor) error {
	if _, ok := exec.(*sql.Tx); ok {
		return errors.New("NEVER: transaction not allowed")
	}
	return nil
}

// NESTED (simulated via SAVEPOINT)
func ProcessRecordNested(ctx context.Context, tx *sql.Tx, r Record) error {
	_, err := tx.ExecContext(ctx, "SAVEPOINT sp_"+r.ID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO records (id, data) VALUES ($1, $2)", r.ID, r.Data)
	if err != nil || r.Data == "fail" {
		_, err = tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT sp_"+r.ID)
		return errors.New("rollback nested: " + r.ID)
	}

	_, err = tx.ExecContext(ctx, "RELEASE SAVEPOINT sp_"+r.ID)
	return err
}

func setupTables(ctx context.Context, db *sql.DB) {
	stmts := []string{
		"CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, name TEXT);",
		"CREATE TABLE IF NOT EXISTS audit_log (user_id TEXT, event TEXT);",
		"CREATE TABLE IF NOT EXISTS records (id TEXT PRIMARY KEY, data TEXT);",
		"CREATE TABLE IF NOT EXISTS config (key TEXT, value TEXT);",
	}
	for _, stmt := range stmts {
		_, err := db.ExecContext(ctx, stmt)
		if err != nil {
			panic(err)
		}
	}

	// after table setup, clean the tables, truncate the tables
	_, err := db.ExecContext(ctx, "TRUNCATE TABLE users, audit_log, records, config;")
	if err != nil {
		panic(err)
	}
}
