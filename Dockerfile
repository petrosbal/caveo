# builder stage
FROM golang:1.26-alpine AS builder
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO_ENABLED=0 for static binary, -ldflags to strip debug info & symbols and overwrite version
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o caveo ./cmd/api



# runner stage
# using google's distroless static image for security
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /app/caveo /caveo
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/caveo"]