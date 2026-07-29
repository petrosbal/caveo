package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
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
		{"GET", "/healthz", app.HandleHealthz, false},
		{"GET", "/readyz", app.HandleReadyz, false},
		{"GET", "/docs", app.handleDocs, false},
		{"GET", "/docs/openapi.yaml", app.handleOpenAPISpec, false},
		{"GET", "/docs/rapidoc-min.js", app.handleRapiDocAsset, false},
	}

	r := chi.NewRouter()

	r.Use(requestID)
	r.Use(app.logRequests)
	r.Use(app.recoverPanics)
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

func (app *Application) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.logger.Enabled(r.Context(), slog.LevelDebug) {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		app.logger.Debug("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}

func (app *Application) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			stacktrace := truncate(string(debug.Stack()), 8192)
			committed := ww.Status() != 0

			app.logger.LogAttrs(r.Context(), slog.LevelError, "panic",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("panic", fmt.Sprint(rec)),
				slog.String("stack", stacktrace),
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.Bool("response_committed", committed),
			)

			if committed {
				panic(http.ErrAbortHandler)
			} else {
				app.respondWithError(ww, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(ww, r)
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newID()
		}
		ctx := context.WithValue(r.Context(), middleware.RequestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var (
	reqIDPrefix  string
	reqIDCounter atomic.Uint64
)

func init() {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	reqIDPrefix = base64.RawURLEncoding.EncodeToString(b)
}

func newID() string {
	return reqIDPrefix + "-" + strconv.FormatUint(reqIDCounter.Add(1), 10)
}
