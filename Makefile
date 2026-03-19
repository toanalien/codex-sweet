.PHONY: fmt fmt-check build test clean

# Format Go code
fmt:
	gofmt -w .

# Check if code is formatted
fmt-check:
	@if [ "$$(gofmt -l . | wc -l)" -gt 0 ]; then \
		echo "Go files are not formatted. Run 'make fmt' to fix:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "All Go files are properly formatted ✓"

# Build binary
build:
	go build -o codex-sweet

# Run tests
test:
	go test -v ./...

# Run vet
vet:
	go vet ./...

# Run all checks
check: fmt-check vet
	@echo "All checks passed ✓"

# Clean build artifacts
clean:
	rm -f codex-sweet
