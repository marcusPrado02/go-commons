module github.com/marcusPrado02/go-commons/examples/inmemory-repository

go 1.24

require (
	github.com/marcusPrado02/go-commons v0.0.0
	github.com/marcusPrado02/go-commons/adapters/persistence/inmemory v0.0.0
)

replace (
	github.com/marcusPrado02/go-commons => ../..
	github.com/marcusPrado02/go-commons/adapters/persistence/inmemory => ../../adapters/persistence/inmemory
)
