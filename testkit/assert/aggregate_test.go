package assert_test

import (
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
	"github.com/marcusPrado02/go-commons/testkit/assert"
)

type orderPlaced struct{ occurredAt time.Time }

func (e orderPlaced) EventType() string     { return "OrderPlaced" }
func (e orderPlaced) OccurredAt() time.Time { return e.occurredAt }

type order struct{ ddd.AggregateRoot[string] }

func TestAssertAggregate_HappyPath(t *testing.T) {
	o := order{AggregateRoot: ddd.NewAggregateRoot("order-1")}
	o.RegisterEvent(orderPlaced{occurredAt: time.Now()})

	assert.AssertAggregate(t, &o).
		HasDomainEvents(1).
		HasEventOfType("OrderPlaced").
		FirstEventSatisfies(func(e ddd.DomainEvent) bool {
			return e.EventType() == "OrderPlaced"
		})
}

func TestAssertAggregate_NoEvents(t *testing.T) {
	o := order{AggregateRoot: ddd.NewAggregateRoot("order-1")}
	assert.AssertAggregate(t, &o).HasNoDomainEvents()
}
