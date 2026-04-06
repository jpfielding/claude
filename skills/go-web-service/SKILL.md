---
name: go-web-service
description: Generate a new Go web service using the established stdlib-only pattern. Use when scaffolding a new cmd/<name>/ service with routes, middleware, args, and graceful shutdown.
---

# Go Web Service Generator

## Instructions

When asked to generate a new Go web service, scaffold three files in `cmd/<name>/` following these conventions exactly.

## File: main.go

```go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"<module>/pkg/core"
	api "<module>/pkg/handlers"
	"<module>/pkg/logging"
)

var GitSHA = "NA"

func main() {
	args := Args{}
	args.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.AppendCtx(ctx, slog.String("git", GitSHA), slog.String("service", "<name>"))

	slog.SetDefault(logging.Logger(os.Stdout, args.LogLevel == "DEBUG", args.SLogLevel()))
	slog.InfoContext(ctx, "init", slog.Any("args", args))

	svc, err := core.NewService(core.Config{ /* ... */ })
	if err != nil {
		slog.ErrorContext(ctx, "initialize service", slog.Any("error", err))
		os.Exit(1)
	}

	r := http.NewServeMux()
	r.HandleFunc("GET /healthz", api.Health(svc))
	// add routes here using Go 1.22+ method+path syntax

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Addr:         args.Address,
		Handler:      withCORS(args.CORSOriginsList(), withRequestLogging(withCompression(r))),
	}

	go func() {
		slog.InfoContext(ctx, "Server Starting", slog.String("address", args.Address), slog.String("log_level", args.LogLevel))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "Server error", slog.Any("error", err))
			cancel()
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigc:
		cancel()
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown", slog.Any("error", err))
	}
	slog.InfoContext(ctx, "Server shutdown")
}
```

## File: args.go

```go
package main

import (
	"flag"
	"log/slog"
	"strings"
)

type Args struct {
	Address     string
	CORSOrigins string
	LogLevel    string
	// add domain-specific flags here
}

func (a *Args) Parse() {
	flag.StringVar(&a.Address, "addr", ":8080", "server listen address (host:port)")
	flag.StringVar(&a.CORSOrigins, "cors-origins", "*", "comma-separated origins (defaults to '*')")
	flag.StringVar(&a.LogLevel, "log-level", "INFO", "log level: DEBUG|INFO|WARN|ERROR")
	flag.Parse()
}

func (a *Args) CORSOriginsList() []string {
	if a.CORSOrigins == "" {
		return []string{"*"}
	}
	return strings.Split(a.CORSOrigins, ",")
}

func (a *Args) SLogLevel() slog.Level {
	switch strings.ToUpper(strings.TrimSpace(a.LogLevel)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

## File: middleware.go

```go
package main

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"<module>/pkg/logging"
)

var allowedMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
	http.MethodOptions,
}

var allowedHeaders = []string{
	"Content-Type",
	"Authorization",
	"User",
}

func withCORS(origins []string, next http.Handler) http.Handler {
	allowAllOrigins := len(origins) == 0 || (len(origins) == 1 && origins[0] == "*")
	allowedMethodsValue := strings.Join(allowedMethods, ", ")
	allowedHeadersValue := strings.Join(allowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowAllOrigins {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && slices.Contains(origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", allowedHeadersValue)
		w.Header().Set("Access-Control-Allow-Methods", allowedMethodsValue)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

func withCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzw := gzip.NewWriter(w)
		defer gzw.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")

		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gzw}, r)
	})
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := logging.AppendCtx(
			r.Context(),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
		)
		r = r.WithContext(ctx)
		slog.DebugContext(ctx, "request started")

		rec := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		slog.DebugContext(ctx, "request completed",
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
```

## Key rules

- **Router**: `http.NewServeMux()` only — no gorilla/chi/echo/gin
- **Logging**: `log/slog` stdlib only
- **Config**: stdlib `flag` package — no viper/cobra
- **Handler shape**: closure returning `http.HandlerFunc`, e.g. `func Health(svc *core.Service) http.HandlerFunc`
- **Middleware chain order**: CORS → request logging → compression → mux
- **Graceful shutdown**: 10-second timeout
- **Build info**: `var GitSHA = "NA"` injected via `-ldflags "-X main.GitSHA=$(git rev-parse HEAD)"`
