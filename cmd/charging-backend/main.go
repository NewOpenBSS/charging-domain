package main

import (
	"context"
	"net/http"
	"os"

	"go-ocs/internal/appl"
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/appcontext"
	graphqlhandler "go-ocs/internal/backend/handlers/graphql"
	"go-ocs/internal/backend/handlers/rest"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"go-ocs/internal/store"

	figure "github.com/common-nighthawk/go-figure"
)

const appName = "Charging Backend"
const defaultConfigFile = "./cmd/charging-backend/backend-config.yaml"

func main() {
	figure.NewFigure(appName, "doom", false).Print()
	logging.Bootstrap()

	configFile := os.Getenv("BACKEND_CONFIG")
	if configFile == "" {
		configFile = defaultConfigFile
	}

	cfg := appcontext.NewConfig(configFile)
	logging.Configure(&cfg.Base.Logging)

	logging.Info("Starting charging-backend", "addr", cfg.Server.Addr)

	db := store.NewStore(cfg.Base.Database.URL)
	defer db.DB.Close()

	kafkaManager := events.ConnectKafka(cfg.Kafkaconfig)
	defer kafkaManager.StopKafka()

	authClient, err := keycloak.NewClient(cfg.Auth)
	if err != nil {
		logging.Fatal("Failed to initialise Keycloak client", "err", err)
	}

	appCtx := appcontext.NewAppContext(cfg, db, kafkaManager, authClient)

	// Start the background tenant resolver so the hostname → wholesaler map stays fresh.
	tenantCtx, tenantCancel := context.WithCancel(context.Background())
	defer tenantCancel()
	appCtx.TenantResolver.Start(tenantCtx)

	// Start the subscriber event consumer so the shadow subscriber table stays in sync.
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	defer appCtx.SubscriberConsumer.Stop()
	appCtx.SubscriberConsumer.Start(consumerCtx)

	metricsSrv := appl.StartMetricsServer(&cfg.Base)
	defer func() {
		if metricsSrv != nil {
			_ = metricsSrv.Shutdown(context.Background())
		}
	}()

	restHandler := rest.NewRouter(appCtx)
	gqlHandler := graphqlhandler.NewRouter(appCtx)

	mux := http.NewServeMux()
	mux.Handle(cfg.Server.GraphqlPath+"/", http.StripPrefix(cfg.Server.GraphqlPath, gqlHandler))
	mux.Handle(cfg.Server.RestPath+"/", http.StripPrefix(cfg.Server.RestPath, restHandler))

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logging.Info("charging-backend listening", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal("Server error", "err", err)
		}
	}()

	sig := appl.WaitForSignal()
	logging.Info("Shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 15*cfg.Server.WriteTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("Graceful shutdown failed", "err", err)
	}

	logging.Info("charging-backend stopped")
}
