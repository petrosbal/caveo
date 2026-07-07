package main

import (
	"encoding/json"
	"net/http"

	"github.com/petrosbal/caveo/internal/hasher"
)

// holds the dependencies for the HTTP handler
type Application struct {
	hasher  *hasher.Service
	limiter *Limiter
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

type ErrorResponse struct {
	Error string `json:"error"`
}

func (app *Application) HandleHash(w http.ResponseWriter, r *http.Request) {
	var req HashRequest

	//decode json body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// basic input validation
	if req.Password == "" {
		app.respondWithError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// call service
	hash, err := app.hasher.Hash(req.Password)
	if err != nil {
		app.respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	// send response
	app.respondWithJSON(w, http.StatusOK, HashResponse{Hash: hash})
}

func (app *Application) HandleVerify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password == "" || req.Hash == "" {
		app.respondWithError(w, http.StatusBadRequest, "Password and hash are required")
		return
	}

	match, err := app.hasher.Verify(req.Password, req.Hash)
	if err != nil {
		app.respondWithError(w, http.StatusBadRequest, "Invalid hash or password")
		return
	}

	app.respondWithJSON(w, http.StatusOK, VerifyResponse{Match: match})
}

func (app *Application) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	app.respondWithError(w, http.StatusNotFound, "Not Found")
}

func (app *Application) respondWithError(w http.ResponseWriter, status int, message string) {
	app.respondWithJSON(w, status, ErrorResponse{Error: message})
}

func (app *Application) respondWithJSON(w http.ResponseWriter, status int, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to encode response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
}
