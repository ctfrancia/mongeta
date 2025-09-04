package worker

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type API struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}
type ErrorResponse struct {
	HTTPStatusCode int
	Message        string
}

func (a *API) Start() {
	a.initRouter()
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}

func (a *API) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", a.StopTaskHandler)
		})
	})
	a.Router.Route("/stats", func(r chi.Router) {
		r.Get("/", a.GetStatsHandler)
	})
}
