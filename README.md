# Caveo: Argon2id Password Hashing Microservice in Go

Caveo is a microservice for password hashing and verification, written in Go. It implements **Argon2id**.
The service is designed as a stateless standalone component to offload cryptographic operations from main application servers.

## Overview

* **Language:** Go 1.25+
* **Algorithm:** Argon2id (`golang.org/x/crypto/argon2`)
* **Deployment:** Docker (Distroless, non-root, multi-stage build).

## Project Structure

```Plaintext
.
├── cmd/api/            # Entry point and HTTP handlers
├── internal/hasher/    # Core Argon2id implementation
├── Dockerfile          # Multi-stage build definition
└── Makefile            # Build and test automation
```

## Design Rationale

**Why a microservice?**
* Argon2id is CPU and memory-intensive. Running these operations on a main API server can degrade throughput for all users. 
* Caveo isolates this resource load, allowing for horizontal scaling and better performance under load.

## Security Implementation

This project adheres to strict cryptographic standards:
1. **Parameter Tuning:** Configured with OWASP parameters (19MB RAM, 2 Iterations, 1 Parallelism). 
2. **Random Salt Generation:** High entropy CSPRNG via crypto/rand instead of typical math/rand.
3. **Timing Attack Mitigation:** `subtle.ConstantTimeCompare` used to prevent timing analysis attacks.
4. **DoS Protection:**
    * 1MB limit on request bodies.
    * Aggressive read/write timeouts (5s/10s) and global request timeout (60s) to prevent resource exhaustion.

## API Reference

The service exposes two endpoints on port `8080`.

### `POST /hash`

Generates a random salt and returns the Argon2id hash.

**Request:**
```json
{ "password": "user_input_password" }
```

**Response:**

```json
{ "hash": "$argon2id$v=19$m=19456,t=2,p=1$..." }
```

### `POST /verify`

Verifies a plaintext password against provided hash string. Derives parameters directly from the hash.

**Request:**

```json
{
  "password": "user_input_password",
  "hash": "$argon2id$v=19$m=19456,t=2,p=1$..."
}
```

**Response (200 OK):**

```json
{ "match": true }
```

### Status Codes

| Code | Description | Reason |
| :--- | :--- | :--- |
| `200` | OK | Operation successful. |
| `400` | Bad Request | Malformed JSON, missing fields, or invalid hash format. |
| `500` | Internal Server Error | Crypto failure or encoding error. |

### Interactive Documentation

* `GET /docs` - browsable API reference, rendered in-browser with RapiDoc.
* `GET /docs/openapi.yaml` - the raw OpenAPI 3.0 spec, for use with API tooling.

## QA - Testing Strategy

Testing covers both unit and integration levels:

* **Unit**: Logic verification for salt uniqueness and vector correctness (`internal/hasher`).

* **Integration**: httptest validation for middleware chains and HTTP contracts (`cmd/api`).

* **Negative Testing**: Explicit coverage for malformed JSON, empty payloads, and invalid hash formats.

## Local Development
### Prerequisites
* **Go** (1.25+)
* **Make**
* **gotestsum** (install via `go install gotest.tools/gotestsum@latest`)

### Commands

The project uses a `Makefile` for standard operations:

```Bash
make run    # Starts the API server locally on :8080
make test   # Runs unit and integration tests 
make build  # Compiles the binary to bin/caveo
make clean  # Removes build artifacts
```

## Docker Deployment

### Container Specifications
The Dockerfile utilizes a multi-stage build process to ensure the final image contains only the binary and necessary root certificates.

* Base Image: `gcr.io/distroless/static-debian12:nonroot`
* User: Runs as non-root (`UID: 65532`)
* Binary: Statically linked with debug symbols stripped (`-ldflags="-s -w"`).


### Build & Run:

```Bash
docker build -t caveo .
docker run -p 8080:8080 caveo
```