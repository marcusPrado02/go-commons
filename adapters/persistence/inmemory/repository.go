// Package inmemory provides a thread-safe, generic in-memory Repository implementation.
// Suitable for unit tests and simple scenarios where a persistent store is not required.
package inmemory

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/marcusPrado02/go-commons/ports/persistence"
)

// SortFunc compares two entities for the given field and direction.
// Return true if a should come before b.
type SortFunc[E any] func(a, b E, field string, descending bool) bool

// options holds optional configuration for InMemoryRepository.
type options[E any] struct {
	sortFunc SortFunc[E]
	ttl      time.Duration
}

// Option configures an InMemoryRepository.
type Option[E any] func(*options[E])

// WithSortFunc provides a comparison function for Search ordering.
func WithSortFunc[E any](fn SortFunc[E]) Option[E] {
	return func(o *options[E]) { o.sortFunc = fn }
}

// WithTTL sets an expiration duration for each saved entity.
// Expired entities are removed by a background goroutine; call Close to stop it.
// Entities are also lazily filtered on read.
func WithTTL[E any](d time.Duration) Option[E] {
	return func(o *options[E]) { o.ttl = d }
}

// InMemoryRepository is a thread-safe, generic repository backed by a map.
// It implements both persistence.Repository and persistence.PageableRepository.
type InMemoryRepository[E any, ID comparable] struct {
	mu          sync.RWMutex
	storage     map[ID]E
	expiry      map[ID]time.Time // non-nil only when TTL is configured
	idExtractor func(E) ID
	opts        options[E]
	stopGC      chan struct{} // non-nil only when TTL is configured
}

// NewInMemoryRepository creates a repository that extracts IDs using idExtractor.
// If WithTTL is provided, a background GC goroutine is started; call Close to stop it.
func NewInMemoryRepository[E any, ID comparable](idExtractor func(E) ID, opts ...Option[E]) *InMemoryRepository[E, ID] {
	o := options[E]{}
	for _, opt := range opts {
		opt(&o)
	}
	r := &InMemoryRepository[E, ID]{
		storage:     make(map[ID]E),
		idExtractor: idExtractor,
		opts:        o,
	}
	if o.ttl > 0 {
		r.expiry = make(map[ID]time.Time)
		r.stopGC = make(chan struct{})
		go r.runGC(o.ttl / 2)
	}
	return r
}

// Close stops the background TTL garbage-collection goroutine, if running.
func (r *InMemoryRepository[E, ID]) Close() {
	if r.stopGC != nil {
		close(r.stopGC)
	}
}

// Clear removes all entities from the repository. Useful for resetting state between tests.
func (r *InMemoryRepository[E, ID]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storage = make(map[ID]E)
	if r.expiry != nil {
		r.expiry = make(map[ID]time.Time)
	}
}

// Save upserts the entity. Returns the saved entity unchanged.
func (r *InMemoryRepository[E, ID]) Save(_ context.Context, entity E) (E, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.idExtractor(entity)
	r.storage[id] = entity
	if r.expiry != nil {
		r.expiry[id] = time.Now().Add(r.opts.ttl)
	}
	return entity, nil
}

// FindByID returns (entity, true, nil) if found, (zero, false, nil) if not found or expired.
func (r *InMemoryRepository[E, ID]) FindByID(_ context.Context, id ID) (E, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.isExpired(id) {
		var zero E
		return zero, false, nil
	}
	entity, ok := r.storage[id]
	return entity, ok, nil
}

// DeleteByID removes the entity. Not an error if not found.
func (r *InMemoryRepository[E, ID]) DeleteByID(_ context.Context, id ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storage, id)
	if r.expiry != nil {
		delete(r.expiry, id)
	}
	return nil
}

// Delete removes the entity by extracting its ID. Not an error if not found.
func (r *InMemoryRepository[E, ID]) Delete(_ context.Context, entity E) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.idExtractor(entity)
	delete(r.storage, id)
	if r.expiry != nil {
		delete(r.expiry, id)
	}
	return nil
}

// FindAll returns a page of entities matching the specification.
func (r *InMemoryRepository[E, ID]) FindAll(ctx context.Context, req persistence.PageRequest, spec persistence.Specification[E]) (persistence.PageResult[E], error) {
	return r.Search(ctx, req, spec, persistence.Sort{})
}

// Search returns a page of entities matching the specification, sorted if a SortFunc is configured.
func (r *InMemoryRepository[E, ID]) Search(_ context.Context, req persistence.PageRequest, spec persistence.Specification[E], s persistence.Sort) (persistence.PageResult[E], error) {
	if err := req.Validate(); err != nil {
		return persistence.PageResult[E]{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	predicate := spec.ToPredicate()
	var matched []E
	for id, entity := range r.storage {
		if r.isExpired(id) {
			continue
		}
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

// isExpired reports whether id has a recorded expiry that has passed.
// Must be called under r.mu (read or write lock).
func (r *InMemoryRepository[E, ID]) isExpired(id ID) bool {
	if r.expiry == nil {
		return false
	}
	exp, ok := r.expiry[id]
	return ok && time.Now().After(exp)
}

func (r *InMemoryRepository[E, ID]) runGC(interval time.Duration) {
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopGC:
			return
		case <-ticker.C:
			r.gc()
		}
	}
}

func (r *InMemoryRepository[E, ID]) gc() {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, exp := range r.expiry {
		if now.After(exp) {
			delete(r.storage, id)
			delete(r.expiry, id)
		}
	}
}
