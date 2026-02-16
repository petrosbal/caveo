# builder stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO_ENABLED=0 for static binary, -ldflags to strip debug info & symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o caveo ./cmd/api



# runner stage
# using google's distroless static image for security
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /
COPY --from=builder /app/caveo /caveo
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/caveo"]