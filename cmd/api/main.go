package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/petrosbal/caveo/internal/hasher"
)

func main() {
	//init service with OWASP defaults
	hashService := hasher.NewService()
	app := &Application{
		hasher: hashService,
	}

	//setup router
	r := chi.NewRouter()

	//standard middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit
			next.ServeHTTP(w, r)
		})
	})

	//expose endpoints
	r.Post("/hash", app.HandleHash)
	r.Post("/verify", app.HandleVerify)

	//start server
	fmt.Printf(`
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

	log.Println("Caveo is listening at port :8080")
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
