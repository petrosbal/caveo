package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/petrosbal/caveo/internal/hasher"
)

// small helper, maybe unnecessary. oh well
func newTestApp() *Application {
	return &Application{
		hasher:  hasher.NewService(),
		limiter: NewLimiter(runtime.GOMAXPROCS(0)), // default limit for tests
	}
}

func TestHandleHash(t *testing.T) {
	app := newTestApp()

	// create a request body (JSON)
	payload := []byte(`{"password": "test_password_123"}`)

	// create a fake HTTP request
	req := httptest.NewRequest(http.MethodPost, "/hash", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	// create a ResponseRecorder (acts like a client)
	w := httptest.NewRecorder()

	// execute the handler
	app.HandleHash(w, req)

	// assertions
	res := w.Result()
	defer res.Body.Close() // this happens last btw

	// check status code
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %v", res.Status)
	}

	// Check response body
	var response HashResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatal("failed to decode response JSON")
	}

	if !strings.HasPrefix(response.Hash, "$argon2id") {
		t.Errorf("expected valid hash, got %s", response.Hash)
	}
}

func TestHandleHash_BadRequest(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "Empty JSON", payload: []byte(`{}`)},
		{name: "Malformed JSON", payload: []byte(`{"password": "ohno`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/hash", bytes.NewBuffer(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.HandleHash(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400 Bad Request, got %d", res.StatusCode)
			}
		})
	}
}

func TestHandleVerify(t *testing.T) {
	app := newTestApp()

	// generate a good hash to test against
	correctPass := "secret"
	validHash, err := app.hasher.Hash(correctPass)
	if err != nil {
		t.Fatalf("failed to generate setup hash: %v", err)
	}

	otherHash, _ := app.hasher.Hash("completely_different_password")

	// define scenarios
	tests := []struct {
		name      string
		inputPass string
		inputHash string
		wantMatch bool
	}{
		{
			name:      "Valid Match",
			inputPass: correctPass,
			inputHash: validHash,
			wantMatch: true,
		},
		{
			name:      "Wrong Password",
			inputPass: "wrong_password",
			inputHash: validHash,
			wantMatch: false,
		},
		{
			name:      "Mismatched Hash",
			inputPass: correctPass,
			inputHash: otherHash,
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// setup payload
			payload := map[string]string{
				"password": tc.inputPass,
				"hash":     tc.inputHash,
			}
			jsonPayload, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(jsonPayload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.HandleVerify(w, req)

			res := w.Result()
			defer res.Body.Close()

			// decode
			var response VerifyResponse
			if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// assert
			if response.Match != tc.wantMatch {
				t.Errorf("expected match=%v, got %v", tc.wantMatch, response.Match)
			}
		})
	}
}

func TestHandleVerify_BadRequest(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "Empty JSON", payload: []byte(`{}`)},
		{name: "Malformed JSON", payload: []byte(`{"hash": "$argon2id$...", "password": "oops`)},
		{name: "Missing Hash Field", payload: []byte(`{"password": "valid"}`)},
		{name: "Missing Password Field", payload: []byte(`{"hash": "some_hash"}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBuffer(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			app.HandleVerify(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400 Bad Request, got %d", res.StatusCode)
			}
		})
	}
}
