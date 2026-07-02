package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/petrosbal/caveo/internal/hasher"
)

func main() {
	//init service with OWASP defaults
	hashService := hasher.NewService()
	app := &Application{
		hasher: hashService,
	}

	port := getEnv("PORT", "8080")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      app.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	printBanner(os.Stdout, port)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func printBanner(w io.Writer, port string) {
	//start server
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
