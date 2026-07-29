package main

import (
	"log/slog"
	"runtime"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{"unset defaults",
			nil,
			Config{
				LogLevel:              slog.LevelInfo,
				MaxConcurrentRequests: runtime.GOMAXPROCS(0),
				DrainDelay:            5 * time.Second,
				Port:                  "8080"},
			false,
		},
		{"all set",
			map[string]string{
				"CAVEO_LOG_LEVEL":               "debug",
				"CAVEO_MAX_CONCURRENT_REQUESTS": "4",
				"CAVEO_DRAIN_DELAY":             "10s",
				"PORT":                          "8081"},
			Config{
				LogLevel:              slog.LevelDebug,
				MaxConcurrentRequests: 4,
				DrainDelay:            10 * time.Second,
				Port:                  "8081"},
			false,
		},
		{"error propagates",
			map[string]string{
				"CAVEO_LOG_LEVEL":               "whatever",
				"CAVEO_MAX_CONCURRENT_REQUESTS": "4",
				"CAVEO_DRAIN_DELAY":             "10s",
				"PORT":                          "8081"},
			Config{},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
			got, err := loadConfig(lookup)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil (value: %v)", got)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	def := slog.LevelInfo
	cases := []struct {
		name    string
		env     map[string]string
		want    slog.Level
		wantErr bool
	}{
		{"unset defaults", nil, def, false},
		{"empty defaults", map[string]string{"CAVEO_LOG_LEVEL": ""}, def, false},
		{"debug value", map[string]string{"CAVEO_LOG_LEVEL": "debug"}, slog.LevelDebug, false},
		{"info value", map[string]string{"CAVEO_LOG_LEVEL": "info"}, slog.LevelInfo, false},
		{"warn value", map[string]string{"CAVEO_LOG_LEVEL": "warn"}, slog.LevelWarn, false},
		{"error value", map[string]string{"CAVEO_LOG_LEVEL": "error"}, slog.LevelError, false},
		{"invalid value", map[string]string{"CAVEO_LOG_LEVEL": "invalid"}, def, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
			got, err := getLogLevel(lookup)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil (value: %s)", got)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("expected %s, got %s", c.want, got)
			}
		})
	}
}

func TestGetConcurrencyLimit(t *testing.T) {
	def := runtime.GOMAXPROCS(0)
	cases := []struct {
		name    string
		env     map[string]string
		want    int
		wantErr bool
	}{
		{"unset defaults", nil, def, false},
		{"empty defaults", map[string]string{"CAVEO_MAX_CONCURRENT_REQUESTS": ""}, def, false},
		{"valid value", map[string]string{"CAVEO_MAX_CONCURRENT_REQUESTS": "4"}, 4, false},
		{"zero rejected", map[string]string{"CAVEO_MAX_CONCURRENT_REQUESTS": "0"}, 0, true},
		{"negative rejected", map[string]string{"CAVEO_MAX_CONCURRENT_REQUESTS": "-1"}, 0, true},
		{"non-integer rejected", map[string]string{"CAVEO_MAX_CONCURRENT_REQUESTS": "abc"}, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
			got, err := getConcurrencyLimit(lookup)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil (value: %d)", got)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("expected %d, got %d", c.want, got)
			}
		})
	}
}

func TestGetDrainDelay(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    float64 // in seconds
		wantErr bool
	}{
		{"unset defaults", nil, 5, false},
		{"empty defaults", map[string]string{"CAVEO_DRAIN_DELAY": ""}, 5, false},
		{"valid value", map[string]string{"CAVEO_DRAIN_DELAY": "10s"}, 10, false},
		{"zero value", map[string]string{"CAVEO_DRAIN_DELAY": "0s"}, 0, false},
		{"minute value", map[string]string{"CAVEO_DRAIN_DELAY": "1m"}, 60, false},
		{"hour value", map[string]string{"CAVEO_DRAIN_DELAY": "1h"}, 3600, false},
		{"millisecond value", map[string]string{"CAVEO_DRAIN_DELAY": "500ms"}, 0.5, false},
		{"microsecond value", map[string]string{"CAVEO_DRAIN_DELAY": "500us"}, 0.0005, false},
		{"negative rejected", map[string]string{"CAVEO_DRAIN_DELAY": "-1s"}, 0, true},
		{"invalid duration rejected", map[string]string{"CAVEO_DRAIN_DELAY": "abc"}, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
			got, err := getDrainDelay(lookup)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil (value: %v)", got)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Seconds() != float64(c.want) {
				t.Errorf("expected %f seconds, got %v seconds", c.want, got.Seconds())
			}
		})
	}
}

func TestGetPort(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"unset defaults", nil, "8080"},
		{"empty defaults", map[string]string{"PORT": ""}, "8080"},
		{"valid value", map[string]string{"PORT": "8081"}, "8081"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup := func(k string) (string, bool) { v, ok := c.env[k]; return v, ok }
			got := getPort(lookup)

			if got != c.want {
				t.Errorf("expected %s, got %s", c.want, got)
			}
		})
	}
}
