# Task 004 — Binary: cmd/charging-housekeeping

**Feature:** F-009 — Charging Domain Housekeeping
**Sequence:** 004 of 004
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Create the `cmd/charging-housekeeping/` binary — a standalone Go program that reads
configuration, connects to the database and Kafka, runs all four housekeeping operations
(quota expiry, stale sessions, trace purge, rate plan cleanup) sequentially, logs a
completion summary, and exits with code 0 on success or non-zero on any error.
This binary is designed to be invoked as a Kubernetes CronJob: it runs all four tasks
once and exits. There is no embedded scheduler.

---

## Scope

**In scope:**
- `cmd/charging-housekeeping/main.go` — application entry point
- `cmd/charging-housekeeping/appcontext/config.go` — `Config` struct and `NewConfig` loader
- `cmd/charging-housekeeping/housekeeping-config.yaml` — minimal local dev config
- Config reads three operational thresholds from environment variables with defaults
  (see Decisions section for rationale re: deviation from YAML-only standard)
- Wire: `store.NewStore`, `events.ConnectKafka`, `quota.NewQuotaManager`, `housekeeping.NewHousekeepingService`
- Run all four tasks sequentially, collect row counts, log structured summary
- Exit code 0 on full success; exit code 1 on any error
- `go build ./...` and `go test -race ./...` must pass

**Out of scope:**
- Helm chart or Kubernetes CronJob YAML (deferred per feature spec)
- Prometheus metrics or alerting on housekeeping outcomes (deferred per feature spec)
- Manual trigger HTTP endpoint (out of scope per feature spec)
- A metrics server (the binary runs and exits; a long-lived metrics endpoint would be meaningless)

---

## Context

**Config design:**
- DB URL, Kafka brokers, and logging come from YAML (same `baseconfig.BaseConfig` + `events.KafkaConfig`
  pattern as other binaries)
- The three operational thresholds are read from environment variables (see Decisions). Defaults:
  - `STALE_SESSIONS_THRESHOLD` — default `24h` (duration string, e.g. `"48h"`, `"30m"`)
  - `TRACE_PURGE_THRESHOLD` — default `36h`
  - `RATEPLAN_CLEANUP_THRESHOLD` — default `720h` (30 days)
- Use `time.ParseDuration(os.Getenv("VAR"))` to parse; fall back to default if env var is absent or empty

**Config struct (`cmd/charging-housekeeping/appcontext/config.go`):**
```go
type Config struct {
    Base        baseconfig.BaseConfig `yaml:"base"`
    Kafkaconfig *events.KafkaConfig   `yaml:"kafka"`
    // Operational thresholds — read from env vars; YAML fields are not populated.
    StaleSessions   time.Duration // STALE_SESSIONS_THRESHOLD; default 24h
    TracePurge      time.Duration // TRACE_PURGE_THRESHOLD;    default 36h
    RatePlanCleanup time.Duration // RATEPLAN_CLEANUP_THRESHOLD; default 720h
}

const (
    defaultStaleSessions   = 24 * time.Hour
    defaultTracePurge      = 36 * time.Hour
    defaultRatePlanCleanup = 30 * 24 * time.Hour
)

func NewConfig(configFilename string) *Config {
    cfg := &Config{
        Kafkaconfig:     events.NewKafkaConfig(),
        StaleSessions:   defaultStaleSessions,
        TracePurge:      defaultTracePurge,
        RatePlanCleanup: defaultRatePlanCleanup,
    }

    if err := baseconfig.LoadConfig(configFilename, cfg); err != nil {
        logging.Fatal("Failed to load config", "err", err)
    }

    // Override thresholds from environment variables if set.
    if v := os.Getenv("STALE_SESSIONS_THRESHOLD"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            cfg.StaleSessions = d
        } else {
            logging.Warn("Invalid STALE_SESSIONS_THRESHOLD; using default", "value", v, "default", defaultStaleSessions)
        }
    }
    if v := os.Getenv("TRACE_PURGE_THRESHOLD"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            cfg.TracePurge = d
        } else {
            logging.Warn("Invalid TRACE_PURGE_THRESHOLD; using default", "value", v, "default", defaultTracePurge)
        }
    }
    if v := os.Getenv("RATEPLAN_CLEANUP_THRESHOLD"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            cfg.RatePlanCleanup = d
        } else {
            logging.Warn("Invalid RATEPLAN_CLEANUP_THRESHOLD; using default", "value", v, "default", defaultRatePlanCleanup)
        }
    }

    return cfg
}
```

**main.go outline:**
```go
const appName = "Charging Housekeeping"
const configFilename = "./cmd/charging-housekeeping/housekeeping-config.yaml"

func main() {
    figure.NewFigure(appName, "doom", false).Print()
    logging.Bootstrap()

    cfg := appcontext.NewConfig(configFilename)
    logging.Init(&cfg.Base.Logging)

    s := store.NewStore(cfg.Base.Database.URL)
    defer s.DB.Close()

    kafkaManager := events.ConnectKafka(cfg.Kafkaconfig)
    defer kafkaManager.StopKafka()

    quotaManager := quota.NewQuotaManager(*s, 3, kafkaManager)
    housekeepingSvc := housekeeping.NewHousekeepingService(s)

    now := time.Now().UTC()
    exitCode := run(context.Background(), now, cfg, s, quotaManager, housekeepingSvc)
    os.Exit(exitCode)
}
```

**`run` function (extracted for testability):**
```go
func run(ctx context.Context, now time.Time, cfg *appcontext.Config, s *store.Store,
         quotaManager *quota.QuotaManager, housekeepingSvc *housekeeping.HousekeepingService) int {

    var (
        quotaCount    int
        sessionsCount int64
        tracesCount   int64
        ratePlanCount int64
        runErr        error
    )

    // Task 1: Quota expiry
    subscribers, err := s.Q.FindExpiredQuotaSubscribers(ctx, pgtype.Timestamptz{Time: now, Valid: true})
    if err != nil {
        logging.Error("Quota expiry: failed to find expired subscribers", "err", err)
        runErr = err
    } else {
        for _, subscriberID := range subscribers {
            if err := quotaManager.ProcessExpiredQuota(ctx, now, subscriberID); err != nil {
                logging.Error("Quota expiry: failed to process subscriber", "subscriberID", subscriberID, "err", err)
                runErr = err
                // continue — process remaining subscribers
            } else {
                quotaCount++
            }
        }
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
        "stale_sessions_deleted",      sessionsCount,
        "traces_deleted",              tracesCount,
        "rate_plan_versions_deleted",  ratePlanCount,
    )

    if runErr != nil {
        return 1
    }
    return 0
}
```

**housekeeping-config.yaml (minimal local dev config):**
```yaml
base:
  appName: charging-housekeeping
  database:
    url: "postgres://ocs:ocs@localhost:5432/ocs?sslmode=disable"
  logging:
    level: "info"
    format: "json"
  metrics:
    enabled: false
    addr: ":9093"
    path: "/metrics"

kafka:
  brokers:
    - "localhost:9092"
  topics:
    quotaJournal: "quota-journal"
```

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Thresholds read from env vars (not YAML) | The Feature spec explicitly requires env vars for the three operational thresholds. For a Kubernetes CronJob, env vars are the standard Kubernetes-native configuration mechanism (injected via CronJob spec). YAML config is appropriate for connection strings and structural settings, not per-environment thresholds. This is a deliberate and documented exception to the YAML-only standard. |
| Default to soft warning (not fatal) on invalid threshold env var | An invalid duration string is most likely a misconfiguration. Using the default allows the job to run with safe defaults rather than failing entirely. The warning is logged so operators are informed. |
| Continue processing remaining quota subscribers after single-subscriber error | A failure for one subscriber (e.g. transient DB error) should not prevent the other subscribers from being processed. All errors are accumulated and the exit code is non-zero if any occurred. |
| Continue all four tasks even if earlier tasks fail | A failure in quota expiry (e.g. Kafka unavailable) should not prevent stale session cleanup. All tasks are independent; the binary reports all failures and exits 1 if any failed. |
| Extract `run()` function for testability | `os.Exit` in `main()` prevents testing. Extracting `run()` allows unit tests to call it directly and inspect the return code without process exit. |
| No Prometheus metrics server | The binary runs and exits (CronJob pattern). A metrics server would only be live for the duration of one run — not useful. Metrics/alerting is explicitly deferred per Feature spec. |

---

## Acceptance Criteria

- [ ] `cmd/charging-housekeeping/main.go` exists and compiles
- [ ] `cmd/charging-housekeeping/appcontext/config.go` reads the three thresholds from env vars with correct defaults
- [ ] `cmd/charging-housekeeping/housekeeping-config.yaml` exists with sensible local dev values
- [ ] All four tasks run sequentially; errors in earlier tasks do not prevent later tasks from running
- [ ] A structured summary log is emitted at completion (all four counts included)
- [ ] Exit code is 0 when all tasks succeed, 1 when any task fails
- [ ] Unit tests for the `run()` function cover: all tasks succeed (exit 0), quota expiry error (exit 1 but other tasks run), at least one cleanup error (exit 1)
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

**Quota expiry:** The highest-risk operation in this binary. `ProcessExpiredQuota` calls
`executeWithQuota` which publishes Kafka journal events and writes to the quota table.
If Kafka is unavailable, events will be lost (not queued). This is an existing limitation
of the quota manager — it is not introduced by this task. Operators should ensure Kafka
is healthy before scheduling this CronJob. If a subscriber's quota processing fails, the
loop continues and the error is logged and reflected in the exit code.

**Rate plan deletion:** Deletes ACTIVE rateplan rows. This is irreversible. The SQL guard
(`AND plan_status = 'ACTIVE'`) and the self-join condition ensure only genuinely superseded
versions are deleted, but operators should validate the `RATEPLAN_CLEANUP_THRESHOLD` before
first production run. A conservative default of 30 days is used.

**Stale session and trace deletion:** Low risk — deletes rows older than a configurable
threshold. These rows are not referenced by other tables (no FK constraints on `charging_data`
or `charging_trace` from other tables). Newer rows are untouched.

---

## Notes

- `os.Exit` must be called only in `main()` — never in library code. The `run()` function
  returns an int and `main()` calls `os.Exit(run(...))`.
- `time.Now().UTC()` is called once in `main()` and passed as `now` to all operations
  (Go standard: never call `time.Now()` inside business logic).
- The configFilename constant assumes the binary is run from the repo root (standard for
  local dev with `go run`). In Kubernetes, `CONFIG_FILE` env var overrides the path
  (via `baseconfig.LoadConfig`).

