package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutesReturnJSON(t *testing.T) {
	app := newTestApp()
	handler := app.Routes()

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantAllow  string
	}{
		{"method not allowed", "GET", "/hash", "", 405, "POST"},
		{"not found", "POST", "/nope", "", 404, ""},
		{"bad request", "POST", "/hash", `{}`, 400, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			handler.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Errorf("want status %d, got %d", c.wantStatus, rec.Code)
			}
			if rec.Header().Get("Content-Type") != "application/json" {
				t.Errorf("want Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
			}
			if c.wantAllow != "" {
				if rec.Header().Get("Allow") != c.wantAllow {
					t.Errorf("want Allow %s, got %s", c.wantAllow, rec.Header().Get("Allow"))
				}
			}
		})
	}
}

func TestDocsRoutesContentType(t *testing.T) {
	app := newTestApp()
	handler := app.Routes()

	cases := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantContent string
	}{
		{"docs page", "GET", "/docs", 200, "text/html"},
		{"openapi spec", "GET", "/docs/openapi.yaml", 200, "application/yaml"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(c.method, c.path, nil)
			handler.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Errorf("want status %d, got %d", c.wantStatus, rec.Code)
			}
			if rec.Header().Get("Content-Type") != c.wantContent {
				t.Errorf("want Content-Type %s, got %s", c.wantContent, rec.Header().Get("Content-Type"))
			}
			if rec.Body.Len() == 0 {
				t.Errorf("want non-empty body, got empty")
			}
		})
	}
}

func TestPanicProducesCorrectError(t *testing.T) {
	app := newTestApp()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hash", strings.NewReader(`{"abc":"def"}`))

	app.recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("want Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var target ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &target); err != nil {
		t.Fatalf("wrong decode response: %v (body: %s)", err, rec.Body.String())
	}
	if target.Error != "Internal server error" {
		t.Errorf("want error %s, got %s", "Internal server error", target.Error)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("want message not to contain %s", "boom")
	}
}

func TestPanicIsLogged(t *testing.T) {
	var buf bytes.Buffer
	app := newTestApp()
	app.logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hash", strings.NewReader(`{"abc":"def"}`))

	app.recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})).ServeHTTP(rec, req)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode log: %v (log: %s)", err, buf.String())
	}

	panicField, ok := got["panic"].(string)
	if !ok {
		t.Fatalf("failed to find panic field in log: %s", buf.String())
	}

	if !strings.Contains(panicField, "boom") {
		t.Errorf("want panic field to contain %q, got %q", "boom", panicField)
	}

	stacktraceField, ok := got["stack"].(string)
	if !ok {
		t.Fatalf("failed to find stack field in log: %s", buf.String())
	}

	if stacktraceField == "" {
		t.Errorf("want stack field to be non-empty, got %q", stacktraceField)
	}

	if got["msg"] != "panic" {
		t.Errorf("want msg field to be %q, got %q", "panic", got["msg"])
	}

	if got["level"] != "ERROR" {
		t.Errorf("want level field to be %q, got %q", "ERROR", got["level"])
	}
}

func TestAbortHandlerPanicPassesThrough(t *testing.T) {
	app := newTestApp()

	var buf bytes.Buffer
	app.logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hash", nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("want middleware to re-panic, got no panic")
		}
		if r != http.ErrAbortHandler {
			t.Errorf("want middleware to re-panic with %v, got %v", http.ErrAbortHandler, r)
		}
		if buf.Len() != 0 {
			t.Errorf("want no log output, got %s", buf.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("want no response body, got %s", rec.Body.String())
		}
	}()

	app.recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})).ServeHTTP(rec, req)
}

func TestPanicAfterResponseStartedAbortsConnection(t *testing.T) {
	app := newTestApp()
	var buf bytes.Buffer
	app.logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		w.(http.Flusher).Flush()
		panic("PANIC_AFTER_WRITE")
	})

	srv := httptest.NewServer(app.recoverPanics(handler))
	defer srv.Close()

	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("header should arrive before panic: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, err = io.ReadAll(resp.Body)
	if err == nil {
		t.Errorf("want body read to fail on aborted connection, got clean read")
	}

	srv.Close()

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode log: %v (log: %s)", err, buf.String())
	}

	committed, ok := got["response_committed"].(bool)
	if !ok {
		t.Fatalf("want response_committed field to be present, got %v", buf.String())
	}
	if !committed {
		t.Errorf("want response_committed field to be true, got %v", committed)
	}

}
