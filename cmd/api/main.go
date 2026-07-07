package main

import (
	"context"
	"fmt"
	"io"
	"log"
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

	app := &Application{
		hasher:  hashService,
		limiter: NewLimiter(limit),
	}
	log.Printf("max concurrent requests: %d", limit)

	port := getEnvString("PORT", "8080")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      app.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	printBanner(os.Stdout, port)

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
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

func printBanner(w io.Writer, port string) {
	fmt.Fprintf(w, `
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
