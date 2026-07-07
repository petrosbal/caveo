package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimiter(t *testing.T) {
	limit := 2
	limiter := NewLimiter(limit)

	if !limiter.TryAcquire() {
		t.Fatal("first acquisition should succeed")
	}
	if !limiter.TryAcquire() {
		t.Fatal("second acquisition should succeed")
	}
	if limiter.TryAcquire() {
		t.Fatal("third acquisition should fail")
	}
	if !limiter.Saturated() {
		t.Fatal("limiter should be saturated")
	}
	limiter.Release()
	if !limiter.TryAcquire() {
		t.Fatal("acquisition after release should succeed")
	}
}

func TestHashShedsWhenSaturated(t *testing.T) {
	app := newTestApp()
	app.limiter = NewLimiter(1)
	app.limiter.TryAcquire() // saturate the limiter

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hash", strings.NewReader(`{"password":"test"}`))
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status code 503, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	if rec.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After header to be 1, got %s", rec.Header().Get("Retry-After"))
	}
}
