package main

import (
	"context"
	"database/sql"
	"fmt"
)

func RunAllScenarios(ctx context.Context, tm *TxManager) {
	fmt.Println("== Running REQUIRED ==")
	err := tm.WithTx(ctx, func(tx *sql.Tx) error {
		return SaveUser(ctx, tx, User{ID: "u1", Name: "Alice"})
	})
	fmt.Println("REQUIRED:", err)

	fmt.Println("== Running SUPPORTS ==")
	err = LogAccess(ctx, tm.DB, "u1")
	fmt.Println("SUPPORTS (no tx):", err)
	tm.WithTx(ctx, func(tx *sql.Tx) error {
		err := LogAccess(ctx, tx, "u1")
		fmt.Println("SUPPORTS (with tx):", err)
		return nil
	})

	fmt.Println("== Running MANDATORY ==")
	err = UpdateProfileMandatoryTx(ctx, tm.DB, User{ID: "u1", Name: "NewName"})
	fmt.Println("MANDATORY (no tx):", err)
	tm.WithTx(ctx, func(tx *sql.Tx) error {
		err := UpdateProfileMandatoryTx(ctx, tx, User{ID: "u1", Name: "InsideTx"})
		fmt.Println("MANDATORY (with tx):", err)
		return nil
	})

	fmt.Println("== Running REQUIRES_NEW ==")
	err = tm.SaveAuditLogRequiresNew(ctx, "u1")
	fmt.Println("REQUIRES_NEW:", err)

	fmt.Println("== Running NOT_SUPPORTED ==")
	err = FetchConfigNotTransactional(ctx, tm.DB)
	fmt.Println("NOT_SUPPORTED (no tx):", err)
	tm.WithTx(ctx, func(tx *sql.Tx) error {
		err := FetchConfigNotTransactional(ctx, tx)
		fmt.Println("NOT_SUPPORTED (with tx):", err)
		return nil
	})

	fmt.Println("== Running NEVER ==")
	err = HealthCheckNeverTransactional(ctx, tm.DB)
	fmt.Println("NEVER (no tx):", err)
	tm.WithTx(ctx, func(tx *sql.Tx) error {
		err := HealthCheckNeverTransactional(ctx, tx)
		fmt.Println("NEVER (with tx):", err)
		return nil
	})

	fmt.Println("== Running NESTED ==")
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		_ = ProcessRecordNested(ctx, tx, Record{ID: "r1", Data: "ok"})
		_ = ProcessRecordNested(ctx, tx, Record{ID: "r2", Data: "failnot"})
		return nil
	})
	fmt.Println("NESTED:", err)
}
