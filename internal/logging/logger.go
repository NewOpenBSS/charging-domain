package logging

import (
	"context"
	"go-ocs/internal/baseconfig"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	levelVar slog.LevelVar
	global   atomic.Value // stores *slog.Logger
)

// Bootstrap initialises a minimal default logger so early startup code can log
// before configuration has been loaded.
//
// Safe to call multiple times; it will only set a default logger if none exists yet.
func Bootstrap() {
	if v := global.Load(); v != nil {
		return
	}

	// Default to JSON + INFO so logs are usable in containers.
	levelVar.Set(slog.LevelDebug)

	opts := &slog.HandlerOptions{
		Level:     &levelVar,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
					src.File = filepath.Base(src.File)
					return slog.Any(slog.SourceKey, src)
				}
			}
			return a
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, opts))
	global.Store(l)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetDefault(l)
}

// Configure applies the configured logging settings (format, level, etc.).
// Intended to be called after configuration has been loaded.
//
// It is safe to call multiple times; the global logger will be replaced.
func Configure(config *baseconfig.LoggingConfig) {
	if config == nil {
		Bootstrap()
		return
	}

	levelVar.Set(parseLevel(config.Level))

	var w io.Writer = os.Stdout

	opts := &slog.HandlerOptions{
		Level:     &levelVar,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Shorten `source` from full path to just `file:line`.
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
					src.File = filepath.Base(src.File)
					return slog.Any(slog.SourceKey, src)
				}
			}
			return a
		},
	}

	var h slog.Handler
	if strings.EqualFold(config.Format, "text") {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}

	l := slog.New(h)
	global.Store(l)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	slog.SetDefault(l)
}

// Init is kept for backward compatibility. Prefer Bootstrap() early, then Configure().
func Init(config *baseconfig.LoggingConfig) {
	Configure(config)
}

func logWithCaller(level slog.Level, msg string, args ...any) {
	l := internalLog()

	// Skip frames: runtime.Callers -> logWithCaller -> Info/Error/... -> user call site
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	// Convert key/value pairs into slog.Attr.
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = "!BADKEY"
		}
		var val any
		if i+1 < len(args) {
			val = args[i+1]
		} else {
			val = "!MISSING"
		}
		r.AddAttrs(slog.Any(key, val))
	}

	_ = l.Handler().Handle(context.Background(), r)
}

func Info(msg string, args ...any) {
	logWithCaller(slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...any) {
	logWithCaller(slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	logWithCaller(slog.LevelError, msg, args...)
}

func Fatal(msg string, args ...any) {
	logWithCaller(slog.LevelError, msg, args...)
	os.Exit(1)
}

func Debug(msg string, args ...any) {
	logWithCaller(slog.LevelDebug, msg, args...)
}

func IsDebug() bool {
	return levelVar.Level() <= slog.LevelDebug
}

// L returns the global logger. Safe even if Init wasn’t called yet.
func internalLog() *slog.Logger {
	if v := global.Load(); v != nil {
		return v.(*slog.Logger)
	}
	Bootstrap()
	return global.Load().(*slog.Logger)
}

// SetLevel allows runtime level changes.
func SetLevel(level string) {
	levelVar.Set(parseLevel(level))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Optional: attach logger to context (useful for request/session scoped fields)
type ctxKey struct{}

func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

func From(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok {
			return l
		}
	}
	return internalLog()
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		start := time.Now()
		defer func() {
			dur := time.Since(start)

			reqID := middleware.GetReqID(r.Context())
			remoteIP := r.RemoteAddr // RealIP middleware will already have fixed this

			// Some folks prefer r.URL.Path, but RouteContext gives you the matched route pattern (e.g. /users/{id})
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					route = p
				}
			}

			status := ww.Status()
			bytes := ww.BytesWritten()

			// Choose level based on status
			if status >= 500 {
				Error("http request",
					"method", r.Method,
					"path", route,
					"status", status,
					"bytes", bytes,
					"duration", dur.String(),
					"req_id", reqID,
					"remote_ip", remoteIP,
				)
			} else {
				Info("http request",
					"method", r.Method,
					"path", route,
					"status", status,
					"bytes", bytes,
					"duration", dur.String(),
					"req_id", reqID,
					"remote_ip", remoteIP,
				)
			}
		}()

		next.ServeHTTP(ww, r)
	})
}
