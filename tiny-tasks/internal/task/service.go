package task

import "tiny-tasks/internal/model"

type Service struct {
	repo TaskRepository
}

func NewService(repo TaskRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(title string) (model.Task, error) {
	valid, err := ValidateTitle(title)
	if err != nil {
		return model.Task{}, err
	}
	return s.repo.Create(valid)
}

func (s *Service) List() ([]model.Task, error) {
	return s.repo.List()
}

func (s *Service) Get(id string) (model.Task, error) {
	return s.repo.Get(id)
}

func (s *Service) Complete(id string) (model.Task, error) {
	completed := true
	return s.repo.Update(id, nil, &completed)
}

func (s *Service) Undo(id string) (model.Task, error) {
	completed := false
	return s.repo.Update(id, nil, &completed)
}

func (s *Service) Patch(id string, title *string, completed *bool) (model.Task, error) {
	if title == nil && completed == nil {
		return model.Task{}, ErrNoFieldsToPatch
	}

	if title != nil {
		valid, err := ValidateTitle(*title)
		if err != nil {
			return model.Task{}, err
		}
		title = &valid
	}

	return s.repo.Update(id, title, completed)
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}
