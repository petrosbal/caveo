package main

import "net/http"

type Limiter struct {
	sem chan struct{}
}

func NewLimiter(limit int) *Limiter {
	return &Limiter{sem: make(chan struct{}, limit)}
}

func (l *Limiter) TryAcquire() bool {
	select {
	case l.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *Limiter) Release() { <-l.sem }

func (l *Limiter) Saturated() bool {
	return len(l.sem) == cap(l.sem)
}

func (app *Application) shedWhenSaturated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.limiter.TryAcquire() {
			w.Header().Set("Retry-After", "1")
			app.respondWithError(w, http.StatusServiceUnavailable, "Server at capacity, please try again later")
			return
		}
		defer app.limiter.Release()
		next.ServeHTTP(w, r)
	})
}
