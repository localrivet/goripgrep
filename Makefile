.PHONY: all build test lint clean bench coverage install cli build-all build-linux build-darwin build-windows clean-dist release

# Default target
all: build test lint

# Build the library
build:
	go build -v ./...

# Build the CLI tool
cli:
	go build -o goripgrep ./cmd/goripgrep

# Install the CLI tool globally
install: cli
	go install ./cmd/goripgrep

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	go clean
	rm -f goripgrep
	rm -rf dist/

# Run benchmarks
bench:
	go test -bench=. -benchmem

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Build for multiple platforms
build-all: clean-dist
	@echo "Building cross-platform binaries..."
	@mkdir -p dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64 dist/windows-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/goripgrep ./cmd/goripgrep
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/goripgrep ./cmd/goripgrep
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/goripgrep ./cmd/goripgrep
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/goripgrep ./cmd/goripgrep
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/goripgrep.exe ./cmd/goripgrep
	GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/goripgrep.exe ./cmd/goripgrep
	@echo "Cross-platform binaries built in dist/ directory"

build-linux:
	@mkdir -p dist/linux-amd64 dist/linux-arm64
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/goripgrep ./cmd/goripgrep
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/goripgrep ./cmd/goripgrep

build-darwin:
	@mkdir -p dist/darwin-amd64 dist/darwin-arm64
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/goripgrep ./cmd/goripgrep
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/goripgrep ./cmd/goripgrep

build-windows:
	@mkdir -p dist/windows-amd64 dist/windows-arm64
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/goripgrep.exe ./cmd/goripgrep
	GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/goripgrep.exe ./cmd/goripgrep

# Clean dist directory
clean-dist:
	rm -rf dist/

# Release build with version and archives
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make release VERSION=v1.0.0"; exit 1; fi
	@echo "Building release $(VERSION)..."
	@mkdir -p dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64 dist/windows-arm64
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/linux-amd64/goripgrep ./cmd/goripgrep
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/linux-arm64/goripgrep ./cmd/goripgrep
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/darwin-amd64/goripgrep ./cmd/goripgrep
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/darwin-arm64/goripgrep ./cmd/goripgrep
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/windows-amd64/goripgrep.exe ./cmd/goripgrep
	GOOS=windows GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o dist/windows-arm64/goripgrep.exe ./cmd/goripgrep
	@echo "Creating archives..."
	@cd dist/linux-amd64 && tar -czf ../goripgrep-$(VERSION)-linux-amd64.tar.gz goripgrep
	@cd dist/linux-arm64 && tar -czf ../goripgrep-$(VERSION)-linux-arm64.tar.gz goripgrep
	@cd dist/darwin-amd64 && tar -czf ../goripgrep-$(VERSION)-darwin-amd64.tar.gz goripgrep
	@cd dist/darwin-arm64 && tar -czf ../goripgrep-$(VERSION)-darwin-arm64.tar.gz goripgrep
	@cd dist/windows-amd64 && zip ../goripgrep-$(VERSION)-windows-amd64.zip goripgrep.exe
	@cd dist/windows-arm64 && zip ../goripgrep-$(VERSION)-windows-arm64.zip goripgrep.exe
	@echo "Release $(VERSION) built successfully!" 