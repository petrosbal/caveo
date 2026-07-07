package main

import (
	"runtime"
	"testing"
)

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
