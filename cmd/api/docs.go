package main

import (
	_ "embed"
	"net/http"
)

//go:embed docs/openapi.yaml
var openAPISpec []byte

//go:embed docs/rapidoc-min.js
var rapidocJS []byte

const docsHTML = `
<!DOCTYPE html>
<html>
  <head>
	<meta charset="UTF-8" />
	<title>Caveo API Documentation</title>
	<script type="module" src="/docs/rapidoc-min.js"></script>
  </head>
  <body>
	<rapi-doc
	  spec-url="/docs/openapi.yaml"
	  theme="dark"
	  show-header="false"
	  render-style="read">
	</rapi-doc>
  </body>
</html>
`

func (app *Application) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(docsHTML))
}

func (app *Application) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	w.Write(openAPISpec)
}

func (app *Application) handleRapiDocAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	w.Write(rapidocJS)
}
