package ddd_test

import (
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent is a minimal DomainEvent implementation for tests.
type testEvent struct {
	eventType  string
	occurredAt time.Time
}

func (e testEvent) EventType() string     { return e.eventType }
func (e testEvent) OccurredAt() time.Time { return e.occurredAt }

// testAggregate embeds AggregateRoot for testing.
type testAggregate struct {
	ddd.AggregateRoot[string]
}

func TestAggregateRoot_ID(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	assert.Equal(t, "agg-1", agg.ID())
}

func TestAggregateRoot_RegisterAndPull(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}

	evt := testEvent{eventType: "OrderPlaced", occurredAt: time.Now()}
	agg.RegisterEvent(evt)

	events := agg.PullDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "OrderPlaced", events[0].EventType())
}

func TestAggregateRoot_PullClearsEvents(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "Evt", occurredAt: time.Now()})

	agg.PullDomainEvents()
	second := agg.PullDomainEvents()

	assert.Empty(t, second)
}

func TestAggregateRoot_PullReturnsCopy(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "Evt", occurredAt: time.Now()})

	events := agg.PullDomainEvents()
	// Mutate the returned slice — should not affect the aggregate
	events[0] = testEvent{eventType: "Mutated", occurredAt: time.Now()}

	agg.RegisterEvent(testEvent{eventType: "Second", occurredAt: time.Now()})
	second := agg.PullDomainEvents()
	require.Len(t, second, 1)
	assert.Equal(t, "Second", second[0].EventType())
}

func TestAggregateRoot_NoEventsInitially(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	assert.Empty(t, agg.PullDomainEvents())
}

func TestAggregateRoot_MultipleEvents(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "A", occurredAt: time.Now()})
	agg.RegisterEvent(testEvent{eventType: "B", occurredAt: time.Now()})
	agg.RegisterEvent(testEvent{eventType: "C", occurredAt: time.Now()})

	events := agg.PullDomainEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "A", events[0].EventType())
	assert.Equal(t, "B", events[1].EventType())
	assert.Equal(t, "C", events[2].EventType())
}
