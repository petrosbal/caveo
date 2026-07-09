package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/petrosbal/caveo/internal/hasher"
)

func main() {
	hashService := hasher.NewService()

	limit, err := getConcurrencyLimit(os.LookupEnv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	drainDelay, err := getDrainDelay(os.LookupEnv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	app := &Application{
		hasher:  hashService,
		limiter: NewLimiter(limit),
	}
	log.Printf("max concurrent requests: %d", limit)
	log.Printf("drain delay: %v", drainDelay)

	port := getEnvString("PORT", "8080")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      app.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}

	app.ready.Store(true)

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	printBanner(os.Stdout, port)

	<-ctx.Done()
	log.Println("shutdown: signal received")
	stop()

	app.ready.Store(false)
	log.Printf("shutdown: marked as not ready")

	if drainDelay > 0 {
		log.Printf("shutdown: waiting %v for deregistration", drainDelay)
		time.Sleep(drainDelay)
	}

	log.Println("shutdown: draining in-flight requests")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: drain incomplete with error: %v", err)
	} else {
		log.Println("shutdown: complete")
	}

}

func getEnvString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
	if d < 0 {
		return 0, fmt.Errorf("CAVEO_DRAIN_DELAY must be a non-negative duration, got: %q", v)
	}
	if err != nil {
		return 0, fmt.Errorf("CAVEO_DRAIN_DELAY must be a valid duration, got: %q", v)
	}
	return d, nil
}

func printBanner(w io.Writer, port string) {
	_, _ = fmt.Fprintf(w, `
   ______                     
  / ____/___ __   _____  ____ 
 / /   / __ `+"`"+`/ | / / _ \/ __ \
/ /___/ /_/ /| |/ /  __/ /_/ /
\____/\__,_/ |___/\___/\____/ v1.0
                              
   Argon2id Microservice
   ---------------------
   Status:  %sONLINE%s
   Controls: Ctrl + C to stop

   --- LIVE LOGS ---
	
`, "\033[32m", "\033[0m")

	log.Println("Caveo is listening at port: ", port)
}
