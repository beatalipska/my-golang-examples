package task

import "tiny-tasks/internal/model"

type TaskRepository interface {
	Create(title string) (model.Task, error)
	List() ([]model.Task, error)
	Get(id string) (model.Task, error)
	Update(id string, title *string, completed *bool) (model.Task, error)
	Delete(id string) error
}
