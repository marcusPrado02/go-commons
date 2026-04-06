// Package persistence defines repository port interfaces following DDD patterns.
// All methods accept context.Context as the first parameter.
package persistence

import "context"

// Repository is the base CRUD port for a domain entity E identified by ID.
// Save is an upsert — it may modify the entity (e.g. assign a generated ID or update timestamps).
type Repository[E any, ID any] interface {
	// Save persists the entity. Returns the saved entity (may differ from input).
	Save(ctx context.Context, entity E) (E, error)
	// FindByID returns (entity, true, nil) if found, (zero, false, nil) if not found,
	// or (zero, false, err) if a technical error occurred.
	FindByID(ctx context.Context, id ID) (E, bool, error)
	// DeleteByID removes the entity with the given ID. Not an error if not found.
	DeleteByID(ctx context.Context, id ID) error
	// Delete removes the entity. Not an error if not found.
	Delete(ctx context.Context, entity E) error
}

// PageableRepository extends Repository with paginated query support.
type PageableRepository[E any, ID any] interface {
	Repository[E, ID]
	// FindAll returns a page of entities matching the specification.
	FindAll(ctx context.Context, req PageRequest, spec Specification[E]) (PageResult[E], error)
	// Search returns a page of entities matching the specification, ordered by sort.
	Search(ctx context.Context, req PageRequest, spec Specification[E], sort Sort) (PageResult[E], error)
}

// Specification filters entities. Use Spec() for simple func-based specs.
// Implement the interface directly for specs that need SQL or Elasticsearch translation.
type Specification[E any] interface {
	// ToPredicate returns an in-memory filter function. Used by InMemoryRepository.
	ToPredicate() func(E) bool
}

// funcSpec wraps a plain function as a Specification.
type funcSpec[E any] struct{ fn func(E) bool }

func (s funcSpec[E]) ToPredicate() func(E) bool { return s.fn }

// Spec creates a Specification from a plain filter function.
// Use for in-memory and test scenarios. For production SQL/ES, implement Specification directly.
func Spec[E any](fn func(E) bool) Specification[E] {
	return funcSpec[E]{fn: fn}
}

// Sort defines the ordering for paginated queries.
type Sort struct {
	Field      string
	Descending bool
}

// PageRequest specifies which page to fetch. Pages are zero-indexed.
type PageRequest struct {
	Page int
	Size int
}

// PageResult is a paginated response containing a slice of entities.
type PageResult[E any] struct {
	Content       []E
	TotalElements int
	TotalPages    int
	Page          int
	Size          int
}
