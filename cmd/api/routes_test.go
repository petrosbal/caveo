package main

import (
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
