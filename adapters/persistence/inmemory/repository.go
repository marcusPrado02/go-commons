// Package inmemory provides a thread-safe, generic in-memory Repository implementation.
// Suitable for unit tests and simple scenarios where a persistent store is not required.
package inmemory

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/marcusPrado02/go-commons/ports/persistence"
)

// SortFunc compares two entities for the given field and direction.
// Return true if a should come before b.
type SortFunc[E any] func(a, b E, field string, descending bool) bool

// options holds optional configuration for InMemoryRepository.
type options[E any] struct {
	sortFunc SortFunc[E]
}

// Option configures an InMemoryRepository.
type Option[E any] func(*options[E])

// WithSortFunc provides a comparison function for Search ordering.
func WithSortFunc[E any](fn SortFunc[E]) Option[E] {
	return func(o *options[E]) { o.sortFunc = fn }
}

// InMemoryRepository is a thread-safe, generic repository backed by a map.
// It implements both persistence.Repository and persistence.PageableRepository.
type InMemoryRepository[E any, ID comparable] struct {
	mu          sync.RWMutex
	storage     map[ID]E
	idExtractor func(E) ID
	opts        options[E]
}

// NewInMemoryRepository creates a repository that extracts IDs using idExtractor.
func NewInMemoryRepository[E any, ID comparable](idExtractor func(E) ID, opts ...Option[E]) *InMemoryRepository[E, ID] {
	o := options[E]{}
	for _, opt := range opts {
		opt(&o)
	}
	return &InMemoryRepository[E, ID]{
		storage:     make(map[ID]E),
		idExtractor: idExtractor,
		opts:        o,
	}
}

// Save upserts the entity. Returns the saved entity unchanged.
func (r *InMemoryRepository[E, ID]) Save(_ context.Context, entity E) (E, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.idExtractor(entity)
	r.storage[id] = entity
	return entity, nil
}

// FindByID returns (entity, true, nil) if found, (zero, false, nil) if not found.
func (r *InMemoryRepository[E, ID]) FindByID(_ context.Context, id ID) (E, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entity, ok := r.storage[id]
	return entity, ok, nil
}

// DeleteByID removes the entity. Not an error if not found.
func (r *InMemoryRepository[E, ID]) DeleteByID(_ context.Context, id ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, id)
	return nil
}

// Delete removes the entity by extracting its ID. Not an error if not found.
func (r *InMemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, r.idExtractor(entity))
	return nil
}

// FindAll returns a page of entities matching the specification.
func (r *InMemoryRepository[E, ID]) FindAll(ctx context.Context, req persistence.PageRequest, spec persistence.Specification[E]) (persistence.PageResult[E], error) {
	return r.Search(ctx, req, spec, persistence.Sort{})
}

// Search returns a page of entities matching the specification, sorted if a SortFunc is configured.
func (r *InMemoryRepository[E, ID]) Search(_ context.Context, req persistence.PageRequest, spec persistence.Specification[E], s persistence.Sort) (persistence.PageResult[E], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	predicate := spec.ToPredicate()
	var matched []E
	for _, entity := range r.storage {
		if predicate(entity) {
			matched = append(matched, entity)
		}
	}

	if r.opts.sortFunc != nil && s.Field != "" {
		sortFn := r.opts.sortFunc
		sort.Slice(matched, func(i, j int) bool {
			return sortFn(matched[i], matched[j], s.Field, s.Descending)
		})
	}

	total := len(matched)
	totalPages := 0
	if req.Size > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(req.Size)))
	}

	start := req.Page * req.Size
	if start >= total {
		return persistence.PageResult[E]{
			Content:       []E{},
			TotalElements: total,
			TotalPages:    totalPages,
			Page:          req.Page,
			Size:          req.Size,
		}, nil
	}

	end := start + req.Size
	if end > total {
		end = total
	}

	return persistence.PageResult[E]{
		Content:       matched[start:end],
		TotalElements: total,
		TotalPages:    totalPages,
		Page:          req.Page,
		Size:          req.Size,
	}, nil
}
