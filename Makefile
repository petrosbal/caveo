# Variables
BINARY_NAME=caveo
ENTRY_POINT=./cmd/api
VERSION=$(shell git describe --tags --always --dirty)

all: run
## help: print this help message
.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: run the application
.PHONY: run
run: build
	@echo "Running Caveo..."
	@./bin/$(BINARY_NAME)

## test: run all tests (requires gotestsum)
.PHONY: test
test:
	@echo "Testing..."
	gotestsum --format testname

## build: build the binary
.PHONY: build
build:
	@echo "Building Caveo..."
	go build -ldflags="-X main.version=$(VERSION)" -o bin/$(BINARY_NAME) $(ENTRY_POINT)

## docker-build: build docker image
.PHONY: docker-build
docker-build: test
	@echo "Building Caveo Docker image..."
	docker build --build-arg VERSION=$(VERSION) -t $(BINARY_NAME) .

## docker-run: run docker image
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Caveo Docker image..."
	docker run --rm --name $(BINARY_NAME) -p 8080:8080 $(BINARY_NAME)

## clean: remove binary
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf bin/