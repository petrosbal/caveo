package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) Routes() http.Handler {

	type route struct {
		method  string
		pattern string
		handler http.HandlerFunc
	}

	routes := []route{
		{"POST", "/hash", app.HandleHash},
		{"POST", "/verify", app.HandleVerify},
		{"GET", "/docs", app.handleDocs},
		{"GET", "/docs/openapi.yaml", app.handleOpenAPISpec},
		{"GET", "/docs/rapidoc-min.js", app.handleRapiDocAsset},
	}

	r := chi.NewRouter()

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
	allowed := map[string][]string{}
	for _, rt := range routes {
		r.MethodFunc(rt.method, rt.pattern, rt.handler)
		allowed[rt.pattern] = append(allowed[rt.pattern], rt.method)
	}

	r.NotFound(app.HandleNotFound)
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		if methods, ok := allowed[req.URL.Path]; ok {
			w.Header().Set("Allow", strings.Join(methods, ", "))
		}
		app.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	return r
}
