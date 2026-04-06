.PHONY: build test lint coverage tidy tidy-all

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

tidy:
	go mod tidy

tidy-all:
	go mod tidy
	@for d in $$(find adapters -name "go.mod" -exec dirname {} \;); do \
		echo "Tidying $$d"; \
		(cd $$d && go mod tidy); \
	done
