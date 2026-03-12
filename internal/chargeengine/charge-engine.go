package chargeengine

import (
	"encoding/json"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/business/interfaces"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

func ChargeEngineHandler(appCtx *appcontext.AppContext) (http.Handler, func()) {

	services, shutdown := engine.NewServiceContext(appCtx)

	r := chi.NewRouter()

	// Common middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Use(logging.Middleware)

	api := appCtx.Config.Engine.Path
	r.Route(api, func(r chi.Router) {
		r.Route("/nchf/chargingdata", func(r chi.Router) {
			// Create charging data
			r.Post("/", nchfHandler(appCtx, services, ProcessCharging))

			// Update / release existing charging data (chargingDataRef is a path param)
			r.Post("/{chargingDataRef}/update", nchfHandler(appCtx, services, UpdateChargingData))
			r.Post("/{chargingDataRef}/release", nchfHandler(appCtx, services, ReleaseChargingData))

			// One-time charge (separate path so it doesn't clash with the create endpoint)
			r.Post("/one-time", nchfHandler(appCtx, services, ProcessOneTimeCharging))
		})
	})

	return r, shutdown
}

// nchfHandler wraps common HTTP concerns (metrics, JSON decode/encode, error handling)
// and delegates the business logic to `process`.
//
// `process` also receives the *http.Request so it can read chi path params.
func nchfHandler(
	appCtx *appcontext.AppContext,
	serviceCtx interfaces.Infrastructure,
	process func(*appcontext.AppContext, interfaces.Infrastructure, string, *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use the matched chi route pattern as the metrics label (stable, not raw URL).
		endpointPath := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if p := rc.RoutePattern(); p != "" {
				endpointPath = p
			}
		}

		// measuring request processing time
		timer := prometheus.NewTimer(appCtx.Metrics.Runtime.WithLabelValues(r.Method, endpointPath))
		defer timer.ObserveDuration()

		// increment request counter
		appCtx.Metrics.Rate.WithLabelValues(r.Method, endpointPath).Inc()

		// decode request body
		var req nchf.ChargingDataRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			appCtx.Metrics.ErrorRate.WithLabelValues(r.Method, endpointPath).Inc()
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json", "detail": err.Error()})
			return
		}

		// process request. Update and Release passes the session as part of the path
		ref := chi.URLParam(r, "chargingDataRef")
		if ref == "" {
			ref = *req.ChargingId
		}
		resp, err := process(appCtx, serviceCtx, ref, &req)

		if err != nil {
			appCtx.Metrics.ErrorRate.WithLabelValues(r.Method, endpointPath).Inc()

			// Check if it's a retransmit error
			if retransmitErr, ok := err.(*ocserrors.RetransmitError); ok {
				writeJSON(w, http.StatusOK, retransmitErr.GetResponse())
				return
			}

			// Keep the response shape simple and consistent
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request failed", "detail": err.Error()})
			return
		}

		// return response
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
