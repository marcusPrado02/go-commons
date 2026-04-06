// Package contracts provides reusable test suites (contracts) for port implementations.
// Embed a contract suite in your adapter tests to verify it satisfies the port contract.
//
// Example:
//
//	func TestInMemoryRepository(t *testing.T) {
//	    suite.Run(t, &contracts.RepositoryContract[User, string]{
//	        Repo:         inmemory.NewInMemoryRepository[User, string](func(u User) string { return u.ID }),
//	        NewEntity:    func() User { return User{ID: uuid.New().String(), Name: "Alice"} },
//	        ExtractID:    func(u User) string { return u.ID },
//	        MutateEntity: func(u User) User { u.Name = "Bob"; return u },
//	    })
//	}
package contracts

import (
	"context"

	"github.com/marcusPrado02/go-commons/ports/persistence"
	"github.com/stretchr/testify/suite"
)

// RepositoryContract is a reusable test suite that verifies a Repository[E, ID] implementation
// satisfies the persistence port contract. Embed it in your adapter test file.
type RepositoryContract[E any, ID comparable] struct {
	suite.Suite
	// Repo is the repository under test. Set before running the suite.
	Repo persistence.Repository[E, ID]
	// NewEntity returns a new, unique entity for each call.
	NewEntity func() E
	// ExtractID extracts the identifier from an entity.
	ExtractID func(E) ID
	// MutateEntity returns a modified copy of the entity (e.g. change a name field).
	MutateEntity func(E) E
}

func (s *RepositoryContract[E, ID]) ctx() context.Context {
	return context.Background()
}

func (s *RepositoryContract[E, ID]) TestSave_InsertsNewEntity() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	id := s.ExtractID(saved)
	found, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.True(ok, "entity should be found after save")
	s.Equal(s.ExtractID(found), id)
}

func (s *RepositoryContract[E, ID]) TestSave_UpdatesExistingEntity() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	mutated := s.MutateEntity(saved)
	updated, err := s.Repo.Save(s.ctx(), mutated)
	s.Require().NoError(err)
	s.Equal(s.ExtractID(saved), s.ExtractID(updated), "ID should not change on update")
}

func (s *RepositoryContract[E, ID]) TestFindByID_Found() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	found, ok, err := s.Repo.FindByID(s.ctx(), s.ExtractID(saved))
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(s.ExtractID(saved), s.ExtractID(found))
}

func (s *RepositoryContract[E, ID]) TestFindByID_NotFound() {
	entity := s.NewEntity()
	id := s.ExtractID(entity)

	_, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.False(ok, "unsaved entity should not be found")
}

func (s *RepositoryContract[E, ID]) TestDeleteByID_Removes() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	id := s.ExtractID(saved)
	err = s.Repo.DeleteByID(s.ctx(), id)
	s.Require().NoError(err)

	_, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.False(ok, "entity should not be found after delete")
}

func (s *RepositoryContract[E, ID]) TestDeleteByID_NotFoundIsNotError() {
	entity := s.NewEntity()
	id := s.ExtractID(entity)
	err := s.Repo.DeleteByID(s.ctx(), id)
	s.NoError(err, "deleting a non-existent entity should not return an error")
}

func (s *RepositoryContract[E, ID]) TestDelete_Removes() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	err = s.Repo.Delete(s.ctx(), saved)
	s.Require().NoError(err)

	_, ok, err := s.Repo.FindByID(s.ctx(), s.ExtractID(saved))
	s.Require().NoError(err)
	s.False(ok)
}
