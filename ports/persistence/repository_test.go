package persistence_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/ports/persistence"
	"github.com/stretchr/testify/assert"
)

func TestSpec_ToPredicate(t *testing.T) {
	spec := persistence.Spec(func(n int) bool { return n > 5 })
	pred := spec.ToPredicate()
	assert.True(t, pred(10))
	assert.False(t, pred(3))
}
