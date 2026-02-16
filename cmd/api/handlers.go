package main

import (
	"encoding/json"
	"net/http"

	"github.com/petrosbal/caveo/internal/hasher"
)

// holds the dependencies for the HTTP handler
type Application struct {
	hasher *hasher.Service
}

type HashRequest struct {
	Password string `json:"password"`
}

type HashResponse struct {
	Hash string `json:"hash"`
}

type VerifyRequest struct {
	Password string `json:"password"`
	Hash     string `json:"hash"`
}

type VerifyResponse struct {
	Match bool `json:"match"`
}

func (app *Application) HandleHash(w http.ResponseWriter, r *http.Request) {
	var req HashRequest

	//decode json body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// basic input validation
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// call service
	hash, err := app.hasher.Hash(req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(HashResponse{Hash: hash}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (app *Application) HandleVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" || req.Hash == "" {
		http.Error(w, "Password and hash are required", http.StatusBadRequest)
		return
	}

	match, err := app.hasher.Verify(req.Password, req.Hash)
	if err != nil {
		http.Error(w, "Invalid hash or password", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(VerifyResponse{Match: match}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
