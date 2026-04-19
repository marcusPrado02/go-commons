package ddd_test

import (
	"fmt"
	"time"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
)

type orderPlaced struct{ occurredAt time.Time }

func (e orderPlaced) OccurredAt() time.Time { return e.occurredAt }
func (e orderPlaced) EventType() string      { return "order.placed" }

type orderID string

type order struct {
	ddd.AggregateRoot[orderID]
}

func newOrder(id orderID) *order {
	o := &order{AggregateRoot: ddd.NewAggregateRoot(id)}
	o.RegisterEvent(orderPlaced{occurredAt: time.Time{}})
	return o
}

func ExampleAggregateRoot_ID() {
	o := newOrder("order-1")
	fmt.Println(o.ID())
	// Output:
	// order-1
}

func ExampleAggregateRoot_PullDomainEvents() {
	o := newOrder("order-2")
	events := o.PullDomainEvents()
	fmt.Println(len(events), events[0].EventType())

	// Pulling again returns nothing — list is cleared
	fmt.Println(len(o.PullDomainEvents()))
	// Output:
	// 1 order.placed
	// 0
}
