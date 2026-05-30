package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ctfrancia/mongeta/logger"
	"github.com/go-chi/chi/v5"
)

type API struct {
	Address      string
	Port         int
	Worker       *Worker
	Router       *chi.Mux
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type ErrorResponse struct {
	HTTPStatusCode int
	Message        string
}

func (a *API) Start(ctx context.Context) {
	a.initRouter()
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", a.Address, a.Port),
		Handler:      a.Router,
		ReadTimeout:  a.ReadTimeout,
		WriteTimeout: a.WriteTimeout,
		IdleTimeout:  a.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("worker HTTP server error", "err", err)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("worker HTTP server shutdown error", "err", err)
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
