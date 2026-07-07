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
		heavy   bool
	}

	routes := []route{
		{"POST", "/hash", app.HandleHash, true},
		{"POST", "/verify", app.HandleVerify, true},
		{"GET", "/docs", app.handleDocs, false},
		{"GET", "/docs/openapi.yaml", app.handleOpenAPISpec, false},
		{"GET", "/docs/rapidoc-min.js", app.handleRapiDocAsset, false},
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
		var h http.Handler = rt.handler
		if rt.heavy {
			h = app.shedWhenSaturated(h)
		}
		r.Method(rt.method, rt.pattern, h)
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
