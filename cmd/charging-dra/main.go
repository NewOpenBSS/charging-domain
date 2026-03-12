package main

import (
	"go-ocs/internal/appl"
	"go-ocs/internal/baseconfig"
	"go-ocs/internal/logging"
	"go-ocs/internal/store"

	"github.com/common-nighthawk/go-figure"
	"github.com/prometheus/client_golang/prometheus"
)

type AppContext struct {
	Config  *Config
	Store   *store.Store
	Ocs     *OcsClient
	Metrics *DRAMetrics
	Limiter *WholesalerLimiterStore
}

const appName = "DRA Server"
const configFilename = "./cmd/charging-dra/dra-config.yaml"

func main() {

	//Init logging
	figure.NewFigure(appName, "doom", false).Print()
	logging.Bootstrap()

	cfg := &Config{}
	err := baseconfig.LoadConfig(configFilename, cfg)
	if err != nil {
		logging.Fatal("Failed to load config", "err", err)
	}

	appContext := &AppContext{Config: cfg}

	// Make sure that the diameter config is valid
	err = validateDiameterFields(&cfg.Diameter)
	if err != nil {
		logging.Fatal("Invalid diameter config", "err", err)
	}

	logging.Init(&cfg.Base.Logging)

	// Create a new OCS client
	appContext.Ocs = NewOcsClient(cfg)

	// create a new store
	appContext.Store = store.NewStore(cfg.Base.Database.URL)

	// The rate limiter
	appContext.Limiter = NewWholesalerLimiterStore()
	//if len(cfg.PProfAddr) > 0 {
	//	go func() {
	//		logging.Info("Starting pprof server", "addr", cfg.PProfAddr)
	//		if err := http.ListenAndServe(cfg.PProfAddr, nil); err != nil {
	//			logging.Error("pprof server error", "err", err)
	//		}
	//	}()
	//}

	appContext.Metrics = NewDRAMetrics(prometheus.DefaultRegisterer)

	//Starting the Metrics server
	metricSvr := appl.StartMetricsServer(&cfg.Base)
	if metricSvr != nil {
		defer metricSvr.Shutdown(nil)
	}

	stop, err := StartDRAServer(appContext)
	if err != nil {
		logging.Fatal("server error ", "err", err)
	}
	defer stop()

	logging.Info("Application ready and to serve... (Ctrl+C to exit)")

	sig := appl.WaitForSignal()
	logging.Info("Graceful shutdown", "signal", sig)
}
