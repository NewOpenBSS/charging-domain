package appl

import (
	"go-ocs/internal/baseconfig"
	"go-ocs/internal/logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(config *baseconfig.BaseConfig) *http.Server {
	if config.Metrics.Enabled {
		r := chi.NewRouter()

		// Common middleware
		r.Use(middleware.RequestID)
		r.Use(middleware.Recoverer)
		r.Use(middleware.Timeout(15 * time.Second))
		r.Use(logging.Middleware)

		r.Get(config.Metrics.Path, promhttp.Handler().ServeHTTP)

		//// Health
		//r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		//	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
		//})

		srv := &http.Server{Addr: config.Metrics.Addr, Handler: r}
		go func() {
			_ = srv.ListenAndServe()
		}()

		logging.Info("Metrics server started", "addr", config.Metrics.Addr, "path", config.Metrics.Path)
		return srv
	}
	return nil
}

func WaitForSignal() os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	sig := <-ch
	return sig
}
