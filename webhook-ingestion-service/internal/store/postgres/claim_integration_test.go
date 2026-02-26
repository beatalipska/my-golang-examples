package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestClaimNextDue_OnlyOneWorkerClaims(t *testing.T) {
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
	id := "evt_claim_once_" + time.Now().UTC().Format("20060102_150405.000000")

	payload := json.RawMessage(`{"type":"payment_succeeded","data":{"x":1}}`)

	created, err := repo.InsertReceived(ctx, id, "payment_succeeded", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatalf("expected inserted event")
	}

	// Run N concurrent claim attempts
	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)

	var mu sync.Mutex
	claimedIDs := make([]string, 0, N)
	errors := make([]error, 0)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			e, ok, err := repo.ClaimNextDue(ctx)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}
			if ok {
				mu.Lock()
				claimedIDs = append(claimedIDs, e.ID)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}

	if len(claimedIDs) != 1 {
		t.Fatalf("expected exactly 1 claim, got %d (%v)", len(claimedIDs), claimedIDs)
	}
	if claimedIDs[0] != id {
		t.Fatalf("expected claimed id %q, got %q", id, claimedIDs[0])
	}

	// sanity: event status should now be processing, attempts=1
	st, attempts, err := repo.GetStatus(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if st != "processing" {
		t.Fatalf("expected status=processing, got %s", st)
	}
	if attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", attempts)
	}
}
