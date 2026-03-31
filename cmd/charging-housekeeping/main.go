package main

import (
	"context"
	"os"
	"time"

	figure "github.com/common-nighthawk/go-figure"

	"go-ocs/cmd/charging-housekeeping/appcontext"
	"go-ocs/internal/events"
	"go-ocs/internal/housekeeping"
	"go-ocs/internal/logging"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
)

const appName = "Charging Housekeeping"
const defaultConfigFile = "./cmd/charging-housekeeping/housekeeping-config.yaml"

func main() {
	figure.NewFigure(appName, "doom", false).Print()
	logging.Bootstrap()

	configFile := os.Getenv("HOUSEKEEPING_CONFIG")
	if configFile == "" {
		configFile = defaultConfigFile
	}

	cfg := appcontext.NewConfig(configFile)
	logging.Configure(&cfg.Base.Logging)

	s := store.NewStore(cfg.Base.Database.URL)
	defer s.DB.Close()

	kafkaManager := events.ConnectKafka(cfg.Kafkaconfig)
	defer kafkaManager.StopKafka()

	quotaManager := quota.NewQuotaManager(*s, 3, kafkaManager)
	housekeepingSvc := housekeeping.NewHousekeepingService(s, quotaManager)

	now := time.Now().UTC()
	exitCode := run(context.Background(), now, cfg, housekeepingSvc)
	os.Exit(exitCode)
}

// run executes all four housekeeping tasks sequentially and returns an exit code.
// Extracted from main() for testability — os.Exit is only called in main().
func run(ctx context.Context, now time.Time, cfg *appcontext.Config,
	housekeepingSvc *housekeeping.HousekeepingService) int {

	var (
		quotaCount    int
		sessionsCount int64
		tracesCount   int64
		ratePlanCount int64
		runErr        error
	)

	// Task 1: Quota expiry
	quotaCount, err := housekeepingSvc.ExpireQuotas(ctx, now)
	if err != nil {
		logging.Error("Quota expiry: failed", "err", err)
		runErr = err
	}

	// Task 2: Stale sessions
	sessionsCount, err = housekeepingSvc.CleanStaleSessions(ctx, now, cfg.StaleSessions)
	if err != nil {
		logging.Error("Stale sessions: failed", "err", err)
		runErr = err
	}

	// Task 3: Trace purge
	tracesCount, err = housekeepingSvc.PurgeOldTraces(ctx, now, cfg.TracePurge)
	if err != nil {
		logging.Error("Trace purge: failed", "err", err)
		runErr = err
	}

	// Task 4: Rate plan cleanup
	ratePlanCount, err = housekeepingSvc.CleanupSupersededRatePlans(ctx, now, cfg.RatePlanCleanup)
	if err != nil {
		logging.Error("Rate plan cleanup: failed", "err", err)
		runErr = err
	}

	// Summary
	logging.Info("Housekeeping complete",
		"quota_subscribers_processed", quotaCount,
		"stale_sessions_deleted", sessionsCount,
		"traces_deleted", tracesCount,
		"rate_plan_versions_deleted", ratePlanCount,
	)

	if runErr != nil {
		return 1
	}
	return 0
}
