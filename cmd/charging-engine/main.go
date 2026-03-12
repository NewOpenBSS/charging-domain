package main

import (
	"go-ocs/internal/appl"
	"go-ocs/internal/chargeengine"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
	"net/http"

	"github.com/common-nighthawk/go-figure"
)

const appName = "Charging Engine"
const configFilename = "./cmd/charging-engine/engine-config.yaml"

func main() {
	figure.NewFigure(appName, "doom", false).Print()
	logging.Bootstrap()

	cfg := appcontext.NewConfig(configFilename)
	logging.Init(&cfg.Base.Logging)

	s := store.NewStore(cfg.Base.Database.URL)

	kafkaManager := events.ConnectKafka(cfg.Kafkaconfig)
	defer kafkaManager.StopKafka()

	appCtx := &appcontext.AppContext{
		Config:       cfg,
		Store:        s,
		Metrics:      appcontext.NewMetrics(),
		QuotaManager: quota.NewQuotaManager(*s, 3, kafkaManager),
		KafkaManager: kafkaManager,
	}

	metricSvr := appl.StartMetricsServer(&cfg.Base)
	defer metricSvr.Shutdown(nil)

	engineStop := StartEngine(appCtx)
	defer engineStop()

	logging.Info("Application ready and to serve... (Ctrl+C to exit)")

	sig := appl.WaitForSignal()
	logging.Info("Graceful shutdown", "signal", sig)
}

func StartEngine(appCtx *appcontext.AppContext) func() {

	handler, shutdownHandler := chargeengine.ChargeEngineHandler(appCtx)
	srv := &http.Server{Addr: appCtx.Config.Engine.Addr, Handler: handler}
	go func() {
		_ = srv.ListenAndServe()
	}()

	logging.Info("Charge Engine server started", "addr", appCtx.Config.Engine.Addr, "path", appCtx.Config.Engine.Path)

	return func() {
		_ = srv.Shutdown(nil)
		shutdownHandler()
	}
}
