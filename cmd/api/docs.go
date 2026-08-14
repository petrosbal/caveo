package main

import (
	_ "embed"
	"net/http"
	"strings"
)

// the docs page is deliberately self-contained. no vendored third party assets,
// and it stays useful with JavaScript disabled (the curl blocks
// are the documentation; the forms are a convenience on top).

//go:embed docs/openapi.yaml
var openAPISpec []byte

//go:embed docs/index.html
var docsTemplate string

// rendered once at startup: the only substitution is the build version.
var docsHTML = strings.ReplaceAll(docsTemplate, "{{VERSION}}", version)

func (app *Application) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// the page loads nothing external. I say so and have the browser enforce it.
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(docsHTML))
}

func (app *Application) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}
