.PHONY: build clean run-historical run-continuous run-stats test deps

# Build the binary
build:
	go build -o bin/analbtc cmd/analbtc/main.go

# Clean build artifacts
clean:
	rm -rf bin/ data/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run historical data ingestion
run-historical: build
	./bin/analbtc -config=config/config.yaml -historical

# Run continuous data ingestion
run-continuous: build
	./bin/analbtc -config=config/config.yaml -continuous

# Show ingestion statistics
run-stats: build
	./bin/analbtc -config=config/config.yaml -stats

# Run full ingestion (historical + continuous)
run: build
	./bin/analbtc -config=config/config.yaml

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Create necessary directories
setup:
	mkdir -p bin data

# Initialize the project
init: setup deps

# Development commands
dev-build: fmt build
dev-run: dev-build run-stats