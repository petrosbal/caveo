package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/petrosbal/caveo/internal/hasher"
)

type Config struct {
	LogLevel              slog.Level
	MaxConcurrentRequests int
	DrainDelay            time.Duration
	Port                  string
}

var version = "dev"

func main() {
	hashService := hasher.NewService()

	var levelVar slog.LevelVar
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: &levelVar,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindDuration {
				a.Value = slog.StringValue(a.Value.Duration().String())
			}
			return a
		},
	}))

	logger.Info("starting", slog.String("version", version))

	cfg, err := loadConfig(os.LookupEnv)
	if err != nil {
		logger.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	levelVar.Set(cfg.LogLevel)

	app := &Application{
		hasher:  hashService,
		limiter: NewLimiter(cfg.MaxConcurrentRequests),
		logger:  logger,
	}

	logger.Info("config",
		slog.String("log_level", cfg.LogLevel.String()),
		slog.Int("max_concurrent_requests", cfg.MaxConcurrentRequests),
		slog.Duration("drain_delay", cfg.DrainDelay),
		slog.String("port", cfg.Port),
	)

	srv := newServer(cfg.Port, app.Routes())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		logger.Error("listen failed", slog.String("port", cfg.Port), slog.String("error", err.Error()))
		os.Exit(1)
	}

	app.ready.Store(true)

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("serve failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	if isTTY(os.Stdout) {
		printBanner(os.Stdout)
	}
	logger.Info("listening", slog.String("port", cfg.Port))

	<-ctx.Done()
	logger.Info("shutdown", slog.String("phase", "signal_received"))
	stop()
	gracefulShutdown(srv, app, cfg.DrainDelay, logger)

}

func loadConfig(lookup func(string) (string, bool)) (Config, error) {
	level, err := getLogLevel(lookup)
	if err != nil {
		return Config{}, err
	}

	limit, err := getConcurrencyLimit(lookup)
	if err != nil {
		return Config{}, err
	}
	drainDelay, err := getDrainDelay(lookup)
	if err != nil {
		return Config{}, err
	}
	port := getPort(lookup)

	return Config{
		LogLevel:              level,
		MaxConcurrentRequests: limit,
		DrainDelay:            drainDelay,
		Port:                  port,
	}, nil
}

func getLogLevel(lookup func(string) (string, bool)) (slog.Level, error) {
	v, ok := lookup("CAVEO_LOG_LEVEL")
	if !ok || v == "" {
		return slog.LevelInfo, nil
	}
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("CAVEO_LOG_LEVEL must be one of: debug, info, warn, error; got: %q", v)
	}
}

func getConcurrencyLimit(lookup func(string) (string, bool)) (int, error) {
	v, ok := lookup("CAVEO_MAX_CONCURRENT_REQUESTS")
	if !ok || v == "" {
		return runtime.GOMAXPROCS(0), nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("CAVEO_MAX_CONCURRENT_REQUESTS must be a positive integer, got: %q", v)
	}
	return n, nil
}

func getDrainDelay(lookup func(string) (string, bool)) (time.Duration, error) {
	v, ok := lookup("CAVEO_DRAIN_DELAY")
	if !ok || v == "" {
		return 5 * time.Second, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("CAVEO_DRAIN_DELAY must be a valid duration, got: %q", v)
	}
	if d < 0 {
		return 0, fmt.Errorf("CAVEO_DRAIN_DELAY must be a non-negative duration, got: %q", v)
	}
	return d, nil
}

func getPort(lookup func(string) (string, bool)) string {
	v, ok := lookup("PORT")
	if !ok || v == "" {
		return "8080"
	}
	return v
}

func newServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func printBanner(w io.Writer) {
	_, _ = fmt.Fprintf(w, `
   ______                     
  / ____/___ __   _____  ____ 
 / /   / __ `+"`"+`/ | / / _ \/ __ \
/ /___/ /_/ /| |/ /  __/ /_/ /
\____/\__,_/ |___/\___/\____/ %s
                              
   Argon2id Microservice
   ---------------------
   Status:  %sONLINE%s
   Controls: Ctrl + C to stop

   --- LIVE LOGS ---
	
`, version, "\033[32m", "\033[0m")
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func gracefulShutdown(srv *http.Server, app *Application, drainDelay time.Duration, logger *slog.Logger) {

	app.ready.Store(false)
	logger.Info("shutdown", slog.String("phase", "not_ready"))

	if drainDelay > 0 {
		logger.Info("shutdown", slog.String("phase", "awaiting_deregistration"), slog.Duration("drain_delay", drainDelay))
		time.Sleep(drainDelay)
	}

	logger.Info("shutdown", slog.String("phase", "draining"))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("shutdown", slog.String("phase", "drain_incomplete"), slog.String("error", err.Error()))
	} else {
		logger.Info("shutdown", slog.String("phase", "complete"))
	}
}
