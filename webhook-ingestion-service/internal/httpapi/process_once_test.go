package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"webhook-ingestion-service/internal/task"
)

type fakeWorkerRepo struct {
	ok       bool
	claimed  bool
	markProc bool
}

func (f *fakeWorkerRepo) ClaimNextDue(ctx context.Context) (task.ClaimedEvent, bool, error) {
	if !f.ok {
		return task.ClaimedEvent{}, false, nil
	}
	f.claimed = true
	return task.ClaimedEvent{ID: "evt_1", Type: "t", Payload: json.RawMessage(`{}`), Attempts: 1}, true, nil
}

func (f *fakeWorkerRepo) MarkProcessed(ctx context.Context, id string) error {
	f.markProc = true
	return nil
}

func (f *fakeWorkerRepo) MarkFailed(ctx context.Context, id string, lastErr string, nextRetryAt time.Time) error {
	return nil
}

type failingProcessor struct{}

func (failingProcessor) Process(ctx context.Context, eventID, eventType string, payload json.RawMessage) error {
	return errors.New("boom")
}

type okProcessor struct{}

func (okProcessor) Process(ctx context.Context, eventID, eventType string, payload json.RawMessage) error {
	return nil
}

func TestProcessOnce_NoWork(t *testing.T) {
	repo := &fakeWorkerRepo{ok: false}
	deps := task.WorkerDeps{Repo: repo, Processor: okProcessor{}}

	h := ProcessOnceHandler(deps)
	req := httptest.NewRequest(http.MethodPost, "/process/once", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Processed"); got != "0" {
		t.Fatalf("X-Processed=%q", got)
	}
}

func TestProcessOnce_ProcessedOne(t *testing.T) {
	repo := &fakeWorkerRepo{ok: true}
	deps := task.WorkerDeps{Repo: repo, Processor: okProcessor{}}

	h := ProcessOnceHandler(deps)
	req := httptest.NewRequest(http.MethodPost, "/process/once", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Processed"); got != "1" {
		t.Fatalf("X-Processed=%q", got)
	}
	if !repo.markProc {
		t.Fatalf("expected MarkProcessed to be called")
	}
}

func TestProcessOnce_Error(t *testing.T) {
	repo := &fakeWorkerRepo{ok: true}
	deps := task.WorkerDeps{Repo: repo, Processor: failingProcessor{}}

	h := ProcessOnceHandler(deps)
	req := httptest.NewRequest(http.MethodPost, "/process/once", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
