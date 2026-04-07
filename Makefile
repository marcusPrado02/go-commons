.PHONY: build test lint coverage tidy tidy-all fmt bench vulncheck mock coverage-report

build:
	go build ./...

test:
	go test ./... -race

lint:
	golangci-lint run ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total

# TOOL-08: Fail if total coverage of root module is below 70%.
coverage-report:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{pct=$$3; sub(/%/,"",pct); if(pct+0 < 70) {print "ERROR: coverage "pct"% is below 70% threshold"; exit 1} else {print "OK: coverage "pct"%"}}'

tidy:
	go mod tidy

tidy-all:
	go mod tidy
	@for d in $$(find adapters -name "go.mod" -exec dirname {} \;); do \
		echo "Tidying $$d"; \
		(cd $$d && go mod tidy); \
	done

# TOOL-02: Format all Go source files.
fmt:
	gofmt -w $$(find . -name "*.go" -not -path "*/vendor/*")
	@if command -v goimports > /dev/null 2>&1; then \
		goimports -w $$(find . -name "*.go" -not -path "*/vendor/*"); \
	fi

# TOOL-03: Run all benchmarks with memory allocation stats.
bench:
	go test -bench=. -benchmem ./...

# TOOL-04: Check for known vulnerabilities in dependencies.
vulncheck:
	@if command -v govulncheck > /dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

# TOOL-05: Generate mocks for all port interfaces.
mock:
	@if command -v mockery > /dev/null 2>&1; then \
		mockery --all --dir ports --output testkit/mocks --outpkg mocks --with-expecter; \
	elif command -v moq > /dev/null 2>&1; then \
		for f in $$(find ports -name "*.go" ! -name "*_test.go"); do \
			moq -out testkit/mocks/$$(basename $$f) $$f; \
		done; \
	else \
		echo "Install mockery: go install github.com/vektra/mockery/v2@latest"; \
		exit 1; \
	fi
