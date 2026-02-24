package httpapi

import (
	"net/http"
	"tiny-tasks/internal/task"
)

type Server struct {
	service *task.Service
	mux     *http.ServeMux
}

func NewServer(service *task.Service) *Server {
	srv := &Server{
		service: service,
		mux:     http.NewServeMux(),
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
