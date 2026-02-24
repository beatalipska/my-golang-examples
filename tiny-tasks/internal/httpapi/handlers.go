package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"tiny-tasks/internal/model"
	"tiny-tasks/internal/task"
)

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

	created, err := s.service.Create(req.Title)
	if err != nil {
		if errors.Is(err, task.ErrInvalidTitle) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	filters, err := parseListFilters(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tasks = filterTasks(tasks, filters)

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
		s.handleGetTask(w, id)
	case http.MethodPatch:
		s.handlePatchTask(w, r, id)
	case http.MethodDelete:
		s.handleDeleteTask(w, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, id string) {
	found, err := s.service.Get(id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, found)
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

	updated, err := s.service.Patch(id, req.Title, req.Completed)
	if err != nil {
		if errors.Is(err, task.ErrInvalidTitle) || errors.Is(err, task.ErrNoFieldsToPatch) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, id string) {
	if err := s.service.Delete(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
