# Variables
BINARY_NAME=caveo
ENTRY_POINT=./cmd/api

all: run
## help: print this help message
.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: run the application
.PHONY: run
run:
	@go run $(ENTRY_POINT)

## test: run all tests (requires gotestsum)
.PHONY: test
test:
	@echo "Testing..."
	gotestsum --format testname

## build: build the binary
.PHONY: build
build:
	@echo "Building Caveo..."
	go build -o bin/$(BINARY_NAME) $(ENTRY_POINT)

## clean: remove binary
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf bin/