package inmemory_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/persistence/inmemory"
	"github.com/marcusPrado02/go-commons/ports/persistence"
	"github.com/marcusPrado02/go-commons/testkit/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type user struct {
	ID   string
	Name string
}

func newRepo() *inmemory.InMemoryRepository[user, string] {
	return inmemory.NewInMemoryRepository[user, string](func(u user) string { return u.ID })
}

// TestInMemoryRepository_Contract runs the shared repository contract suite.
func TestInMemoryRepository_Contract(t *testing.T) {
	counter := 0
	suite.Run(t, &contracts.RepositoryContract[user, string]{
		Repo: newRepo(),
		NewEntity: func() user {
			counter++
			return user{ID: fmt.Sprintf("user-%d", counter), Name: "Alice"}
		},
		ExtractID:    func(u user) string { return u.ID },
		MutateEntity: func(u user) user { u.Name = "Bob"; return u },
	})
}

func TestInMemoryRepository_FindAll_WithSpec(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	_, _ = repo.Save(ctx, user{ID: "1", Name: "Alice"})
	_, _ = repo.Save(ctx, user{ID: "2", Name: "Bob"})
	_, _ = repo.Save(ctx, user{ID: "3", Name: "Alice"})

	result, err := repo.FindAll(ctx,
		persistence.PageRequest{Page: 0, Size: 10},
		persistence.Spec[user](func(u user) bool { return u.Name == "Alice" }),
	)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalElements)
	assert.Len(t, result.Content, 2)
}

func TestInMemoryRepository_FindAll_Pagination(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, _ = repo.Save(ctx, user{ID: fmt.Sprintf("%d", i), Name: "User"})
	}

	result, err := repo.FindAll(ctx,
		persistence.PageRequest{Page: 1, Size: 2},
		persistence.Spec[user](func(user) bool { return true }),
	)
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalElements)
	assert.Equal(t, 3, result.TotalPages) // ceil(5/2)
	assert.Len(t, result.Content, 2)
}

func TestInMemoryRepository_Search_WithSort(t *testing.T) {
	repo := inmemory.NewInMemoryRepository[user, string](
		func(u user) string { return u.ID },
		inmemory.WithSortFunc(func(a, b user, field string, desc bool) bool {
			if field == "Name" {
				if desc {
					return a.Name > b.Name
				}
				return a.Name < b.Name
			}
			return false
		}),
	)
	ctx := context.Background()
	_, _ = repo.Save(ctx, user{ID: "1", Name: "Charlie"})
	_, _ = repo.Save(ctx, user{ID: "2", Name: "Alice"})
	_, _ = repo.Save(ctx, user{ID: "3", Name: "Bob"})

	result, err := repo.Search(ctx,
		persistence.PageRequest{Page: 0, Size: 10},
		persistence.Spec[user](func(user) bool { return true }),
		persistence.Sort{Field: "Name", Descending: false},
	)
	require.NoError(t, err)
	require.Len(t, result.Content, 3)
	assert.Equal(t, "Alice", result.Content[0].Name)
	assert.Equal(t, "Bob", result.Content[1].Name)
	assert.Equal(t, "Charlie", result.Content[2].Name)
}
