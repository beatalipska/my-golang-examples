package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tiny-tasks/internal/httpapi"
	"tiny-tasks/internal/model"
	"tiny-tasks/internal/store"
)

func newTestServer() *httptest.Server {
	st := store.NewTaskStore()
	srv := httpapi.NewServer(st)
	return httptest.NewServer(srv)
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any) (*http.Response, []byte) {
	t.Helper()

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, data
}

func decodeTask(t *testing.T, data []byte) model.Task {
	t.Helper()
	var task model.Task
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("unmarshal task: %v; body=%s", err, string(data))
	}
	return task
}

func decodeList(t *testing.T, data []byte) (int, []model.Task) {
	t.Helper()
	var payload struct {
		Count int          `json:"count"`
		Items []model.Task `json:"items"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal list: %v; body=%s", err, string(data))
	}
	return payload.Count, payload.Items
}

func decodeErr(t *testing.T, data []byte) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v; body=%s", err, string(data))
	}
	return payload.Error
}

func TestHealthz(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, body := doJSON(t, ts.Client(), http.MethodGet, ts.URL+"/healthz", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
}

func TestCreateTask_ValidatesTitle(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	resp, body := doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{
		"title": "  ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	if msg := decodeErr(t, body); msg == "" {
		t.Fatalf("expected error message, got empty")
	}
}

func TestCreateAndGetTask(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create
	resp, body := doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{
		"title": "Buy milk",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	created := decodeTask(t, body)
	if created.ID == "" {
		t.Fatalf("expected id")
	}
	if created.Title != "Buy milk" {
		t.Fatalf("title=%q", created.Title)
	}
	if created.CompletedAt != nil {
		t.Fatalf("expected completed_at to be nil")
	}

	// Get
	resp, body = doJSON(t, ts.Client(), http.MethodGet, ts.URL+"/tasks/"+created.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	got := decodeTask(t, body)
	if got.ID != created.ID {
		t.Fatalf("id mismatch got=%s want=%s", got.ID, created.ID)
	}
}

func TestPatchCompletedTrueFalse(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create
	resp, body := doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{
		"title": "Write tests",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	task := decodeTask(t, body)

	// Mark as done
	resp, body = doJSON(t, ts.Client(), http.MethodPatch, ts.URL+"/tasks/"+task.ID, map[string]any{
		"completed": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	updated := decodeTask(t, body)
	if updated.CompletedAt == nil {
		t.Fatalf("expected completed_at to be set")
	}

	// Mark as not done
	resp, body = doJSON(t, ts.Client(), http.MethodPatch, ts.URL+"/tasks/"+task.ID, map[string]any{
		"completed": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	updated2 := decodeTask(t, body)
	if updated2.CompletedAt != nil {
		t.Fatalf("expected completed_at to be nil after undo")
	}
}

func TestListFiltersCompleted(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create two tasks
	resp, body := doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{"title": "A"})
	if resp.StatusCode != http.StatusBadRequest {
		// "A" is too short; create proper tasks
	}

	resp, body = doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{"title": "Task One"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	t1 := decodeTask(t, body)

	resp, body = doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{"title": "Task Two"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	t2 := decodeTask(t, body)

	// Mark second as completed
	resp, body = doJSON(t, ts.Client(), http.MethodPatch, ts.URL+"/tasks/"+t2.ID, map[string]any{"completed": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	// completed=true -> should return only completed tasks
	resp, body = doJSON(t, ts.Client(), http.MethodGet, ts.URL+"/tasks?completed=true", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	count, items := decodeList(t, body)
	if count != len(items) {
		t.Fatalf("count mismatch count=%d len(items)=%d", count, len(items))
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 completed task, got %d", len(items))
	}
	if items[0].ID != t2.ID {
		t.Fatalf("expected task two, got %s", items[0].ID)
	}

	// completed=false -> should return only not completed tasks
	resp, body = doJSON(t, ts.Client(), http.MethodGet, ts.URL+"/tasks?completed=false", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	_, items = decodeList(t, body)
	if len(items) != 1 {
		t.Fatalf("expected 1 not-completed task, got %d", len(items))
	}
	if items[0].ID != t1.ID {
		t.Fatalf("expected task one, got %s", items[0].ID)
	}
}

func TestListFilterCompletedOn(t *testing.T) {
	ts := newTestServer()
	defer ts.Close()

	// Create and complete a task now (UTC)
	resp, body := doJSON(t, ts.Client(), http.MethodPost, ts.URL+"/tasks", map[string]any{"title": "Complete today"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	task := decodeTask(t, body)

	resp, body = doJSON(t, ts.Client(), http.MethodPatch, ts.URL+"/tasks/"+task.ID, map[string]any{"completed": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	task = decodeTask(t, body)
	if task.CompletedAt == nil {
		t.Fatalf("expected completed_at set")
	}

	// Filter by today's UTC date
	todayUTC := time.Now().UTC().Format("2006-01-02")
	resp, body = doJSON(t, ts.Client(), http.MethodGet, ts.URL+"/tasks?completed_on="+todayUTC, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}
	_, items := decodeList(t, body)
	if len(items) != 1 {
		t.Fatalf("expected 1 task completed today, got %d", len(items))
	}
	if items[0].ID != task.ID {
		t.Fatalf("expected %s, got %s", task.ID, items[0].ID)
	}
}
