package worker

import (
	"context"
	"fmt"
	"log"
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

func (a *API) Start(ctx context.Context) {
	a.initRouter()
	srv := &http.Server{Addr: fmt.Sprintf("%s:%d", a.Address, a.Port), Handler: a.Router}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
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
