package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"tiny-tasks/internal/ids"
	"tiny-tasks/internal/model"
)

var ErrNotFound = errors.New("not found")

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]model.Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]model.Task)}
}

func (s *TaskStore) Create(title string) model.Task {
	now := time.Now().UTC()
	t := model.Task{
		ID:          ids.NewID(),
		Title:       strings.TrimSpace(title),
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: nil,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return t
}

func (s *TaskStore) List() []model.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	return out
}

func (s *TaskStore) Get(id string) (model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	return t, nil
}

func (s *TaskStore) Update(id string, title *string, completed *bool) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}

	if title != nil {
		t.Title = strings.TrimSpace(*title)
	}

	if completed != nil {
		if *completed {
			if t.CompletedAt == nil {
				now := time.Now().UTC()
				t.CompletedAt = &now
			}
		} else {
			t.CompletedAt = nil
		}
	}

	t.UpdatedAt = time.Now().UTC()
	s.tasks[id] = t
	return t, nil
}

func (s *TaskStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
