package main

import (
	"context"
	"database/sql"
	"fmt"
)

func RunAllScenarios(ctx context.Context, tm *TxManager) {
	// Here we have specifically run the save user inside a transaction, in spring boot when we add the @Transaction
	// annotation to a method, it will create a transaction and run the method inside that transaction, no need to
	// create a transaction manually. If there is a transaction already present, it will use that transaction.
	fmt.Println("== Running REQUIRED ==> With Transaction")
	err := tm.WithTx(ctx, func(tx *sql.Tx) error {
		return SaveUser(ctx, tx, User{ID: "u1", Name: "Alice"})
	})
	fmt.Println("REQUIRED:", err)

	// Here in the SUPPORTS scenario, first we have run the LogAccess method without a transaction, and it should work
	// fine. Then we have run the LogAccess method inside a transaction, and it should also work fine.
	fmt.Println("== Running SUPPORTS ==> Without Transaction")
	err = LogAccess(ctx, tm.DB, "u1")
	if err != nil {
		fmt.Println("SUPPORTS with no transaction failed:", err)
	}
	fmt.Println("== Running SUPPORTS ==> With Transaction")
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		err = LogAccess(ctx, tx, "u1")
		return err
	})
	if err != nil {
		fmt.Println("SUPPORTS with transaction failed:", err)
		return
	}

	// Here in the MANDATORY scenario, we have run the UpdateProfileMandatoryTx method without a transaction, and it should
	// fail. Then we have run the UpdateProfileMandatoryTx method inside a transaction, and it should work fine.
	fmt.Println("== Running MANDATORY ==> Without Transaction")
	err = UpdateProfileMandatoryTx(ctx, tm.DB, User{ID: "u1", Name: "NewName"})
	if err != nil {
		fmt.Println("MANDATORY (no tx) failed:", err)
	}
	err = tm.WithTx(ctx, func(tx *sql.Tx) error {
		err := UpdateProfileMandatoryTx(ctx, tx, User{ID: "u1", Name: "InsideTx"})
		return err
	})
	if err != nil {
		fmt.Println("MANDATORY (with tx) failed:", err)
	}

	// Here in the REQUIRES_NEW scenario, we have run the SaveAuditLogRequiresNew method and it will create a new transaction
	// and run the method inside that transaction. If there is a transaction already present, it will create a new transaction.
	fmt.Println("== Running REQUIRES_NEW ==")
	err = tm.SaveAuditLogRequiresNew(ctx, "u1")
	fmt.Println("REQUIRES_NEW:", err)

	//
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
