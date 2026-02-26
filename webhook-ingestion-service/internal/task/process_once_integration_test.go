package task_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	pg "webhook-ingestion-service/internal/store/postgres"
	"webhook-ingestion-service/internal/task"
)

type flakyProcessor struct {
	failFor int
	calls   int
}

func (p *flakyProcessor) Process(ctx context.Context, eventID, eventType string, payload json.RawMessage) error {
	p.calls++
	if p.calls <= p.failFor {
		return errors.New("transient error")
	}
	return nil
}

func TestProcessOnce_FailsTwiceThenProcesses(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set (integration test)")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := pg.NewEventRepo(db)

	ctx := context.Background()
	id := "evt_process_once_" + time.Now().UTC().Format("20060102_150405.000000")

	payload := json.RawMessage(`{"type":"payment_succeeded","data":{"x":1}}`)
	created, err := repo.InsertReceived(ctx, id, "payment_succeeded", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatalf("expected inserted event")
	}

	// deterministic time + rng
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	rng := rand.New(rand.NewSource(1))

	deps := task.WorkerDeps{
		Repo:      repo, // repo must implement WorkerRepository
		Processor: &flakyProcessor{failFor: 2},
		Backoff:   task.BackoffConfig{BaseDelay: 1 * time.Millisecond, MaxDelay: 1 * time.Millisecond},
		RNG:       rng,
		Now:       func() time.Time { return now },
	}

	// 1st attempt: fail
	claimed, err := task.ProcessOnce(ctx, deps)
	if !claimed {
		t.Fatalf("expected claimed=true")
	}
	if err == nil {
		t.Fatalf("expected error on first attempt")
	}

	st, attempts, err2 := repo.GetStatus(ctx, id)
	if err2 != nil {
		t.Fatal(err2)
	}
	if st != "failed" {
		t.Fatalf("expected status=failed after first error, got %s", st)
	}
	if attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", attempts)
	}

	// Make it due again immediately (avoid sleeping)
	mustSetDueNow(t, db, id)

	// 2nd attempt: fail
	claimed, err = task.ProcessOnce(ctx, deps)
	if !claimed {
		t.Fatalf("expected claimed=true")
	}
	if err == nil {
		t.Fatalf("expected error on second attempt")
	}

	st, attempts, err2 = repo.GetStatus(ctx, id)
	if err2 != nil {
		t.Fatal(err2)
	}
	if st != "failed" {
		t.Fatalf("expected status=failed after second error, got %s", st)
	}
	if attempts != 2 {
		t.Fatalf("expected attempts=2, got %d", attempts)
	}

	mustSetDueNow(t, db, id)

	// 3rd attempt: success
	claimed, err = task.ProcessOnce(ctx, deps)
	if !claimed {
		t.Fatalf("expected claimed=true")
	}
	if err != nil {
		t.Fatalf("expected nil on third attempt, got %v", err)
	}

	// verify processed
	ev, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != "processed" {
		t.Fatalf("expected processed, got %s", ev.Status)
	}
	if ev.ProcessedAt == nil {
		t.Fatalf("expected processed_at to be set")
	}
	if ev.Attempts != 3 {
		t.Fatalf("expected attempts=3, got %d", ev.Attempts)
	}
}

func mustSetDueNow(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`UPDATE events SET next_retry_at = now() WHERE id = $1`, id)
	if err != nil {
		t.Fatalf("force due now: %v", err)
	}
}
