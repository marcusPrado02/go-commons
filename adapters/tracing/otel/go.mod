module github.com/marcusPrado02/go-commons/adapters/tracing/otel

go 1.25.0

replace github.com/marcusPrado02/go-commons => ../../..

require (
	github.com/marcusPrado02/go-commons v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
)

require github.com/cespare/xxhash/v2 v2.3.0 // indirect
