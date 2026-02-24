package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"tiny-tasks/internal/ids"
	"tiny-tasks/internal/model"
)

type TaskRepository interface {
	Create(title string) model.Task
	List() []model.Task
	Get(id string) (model.Task, error)
	Update(id string, title *string, completed *bool) (model.Task, error)
	Delete(id string) error
}

type Server struct {
	repo TaskRepository
	mux  *http.ServeMux
}

func NewServer(repo TaskRepository) *Server {
	srv := &Server{
		repo: repo,
		mux:  http.NewServeMux(),
	}

	srv.mux.HandleFunc("GET /healthz", srv.handleHealth)

	srv.mux.HandleFunc("POST /tasks", srv.handleCreateTask)
	srv.mux.HandleFunc("GET /tasks", srv.handleListTasks)

	srv.mux.HandleFunc("/tasks/", srv.handleTaskByID)

	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	withMiddleware(s.mux).ServeHTTP(w, r)
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		r = r.WithContext(ctx)

		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = ids.NewID()
		}
		w.Header().Set("X-Request-Id", reqID)

		next.ServeHTTP(w, r)

		log.Printf("req_id=%s method=%s path=%s dur=%s",
			reqID, r.Method, r.URL.Path, time.Since(start))
	})
}

// ---- handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

type createTaskRequest struct {
	Title string `json:"title"`
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	title := strings.TrimSpace(req.Title)
	if len(title) < 3 {
		writeError(w, http.StatusBadRequest, "title must be at least 3 characters")
		return
	}

	task := s.repo.Create(title)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.repo.List()

	q := r.URL.Query()

	if v := q.Get("completed"); v != "" {
		wantCompleted, err := parseBoolStrict(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "completed must be true or false")
			return
		}
		tasks = filterByCompleted(tasks, wantCompleted)
	}

	if day := q.Get("completed_on"); day != "" {
		start, end, err := parseUTCDayRange(day)
		if err != nil {
			writeError(w, http.StatusBadRequest, "completed_on must be YYYY-MM-DD")
			return
		}
		tasks = filterByCompletedBetween(tasks, start, end)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(tasks),
		"items": tasks,
	})
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTask(w, r, id)
	case http.MethodPatch:
		s.handlePatchTask(w, r, id)
	case http.MethodDelete:
		s.handleDeleteTask(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := s.repo.Get(id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type patchTaskRequest struct {
	Title     *string `json:"title,omitempty"`
	Completed *bool   `json:"completed,omitempty"`
}

func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request, id string) {
	var req patchTaskRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Title == nil && req.Completed == nil {
		writeError(w, http.StatusBadRequest, "provide at least one field: title or completed")
		return
	}

	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if len(t) < 3 {
			writeError(w, http.StatusBadRequest, "title must be at least 3 characters")
			return
		}
		*req.Title = t
	}

	task, err := s.repo.Update(id, req.Title, req.Completed)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.repo.Delete(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- filtering + parsing (kept here for now)

func filterByCompleted(tasks []model.Task, completed bool) []model.Task {
	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		isCompleted := t.CompletedAt != nil
		if isCompleted == completed {
			out = append(out, t)
		}
	}
	return out
}

func filterByCompletedBetween(tasks []model.Task, start, end time.Time) []model.Task {
	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.CompletedAt == nil {
			continue
		}
		ct := *t.CompletedAt
		if !ct.Before(start) && ct.Before(end) {
			out = append(out, t)
		}
	}
	return out
}

func parseBoolStrict(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("not a bool")
	}
}

func parseUTCDayRange(day string) (time.Time, time.Time, error) {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return start, end, nil
}
