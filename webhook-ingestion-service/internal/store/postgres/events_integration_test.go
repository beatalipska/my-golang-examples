package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestInsertReceived_Dedup(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set (integration test)")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewEventRepo(db)

	ctx := context.Background()
	id := "evt_test_dedup_1"
	payload := json.RawMessage(`{"type":"payment_succeeded","data":{"x":1}}`)

	created1, err := repo.InsertReceived(ctx, id, "payment_succeeded", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !created1 {
		t.Fatalf("expected created on first insert")
	}

	created2, err := repo.InsertReceived(ctx, id, "payment_succeeded", payload)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatalf("expected duplicate on second insert")
	}
}
