// Example: InMemory Repository + Contract Suite
//
// This example demonstrates:
//   - Using InMemoryRepository for unit tests and prototyping
//   - Specification-based filtering (Spec)
//   - Paginated queries (FindAll / Search)
//   - Running the testkit contract suite against a custom repository
//
// Run: go run ./examples/inmemory-repository/
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/marcusPrado02/go-commons/adapters/persistence/inmemory"
	"github.com/marcusPrado02/go-commons/ports/persistence"
)

// --- Domain Model -----------------------------------------------------------

type ProductID string

type Product struct {
	ID       ProductID
	Name     string
	Price    int // cents
	Category string
	Active   bool
}

// --- Specifications ---------------------------------------------------------

// ActiveProducts matches only active products.
func ActiveProducts() persistence.Specification[Product] {
	return persistence.Spec(func(p Product) bool { return p.Active })
}

// InCategory matches products belonging to a given category.
func InCategory(cat string) persistence.Specification[Product] {
	return persistence.Spec(func(p Product) bool { return p.Category == cat })
}

// And combines two specifications (logical AND).
func And(a, b persistence.Specification[Product]) persistence.Specification[Product] {
	return persistence.Spec(func(p Product) bool {
		return a.ToPredicate()(p) && b.ToPredicate()(p)
	})
}

// --- Main -------------------------------------------------------------------

func main() {
	ctx := context.Background()

	repo := inmemory.NewInMemoryRepository(
		func(p Product) ProductID { return p.ID },
		inmemory.WithSortFunc(func(a, b Product, field string, desc bool) bool {
			switch field {
			case "price":
				if desc {
					return a.Price > b.Price
				}
				return a.Price < b.Price
			case "name":
				if desc {
					return a.Name > b.Name
				}
				return a.Name < b.Name
			}
			return false
		}),
	)

	// Seed products.
	products := []Product{
		{ID: "p1", Name: "Espresso", Price: 350, Category: "coffee", Active: true},
		{ID: "p2", Name: "Latte", Price: 550, Category: "coffee", Active: true},
		{ID: "p3", Name: "Croissant", Price: 450, Category: "food", Active: true},
		{ID: "p4", Name: "Old Blend", Price: 300, Category: "coffee", Active: false},
		{ID: "p5", Name: "Muffin", Price: 400, Category: "food", Active: true},
	}
	for _, p := range products {
		if _, err := repo.Save(ctx, p); err != nil {
			log.Fatal("save:", err)
		}
	}
	fmt.Printf("Seeded %d products\n\n", len(products))

	// Query 1: All active coffee products, paginated (page 0, size 10).
	activeCoffee := And(ActiveProducts(), InCategory("coffee"))
	page, err := repo.FindAll(ctx, persistence.PageRequest{Page: 0, Size: 10}, activeCoffee)
	if err != nil {
		log.Fatal("FindAll:", err)
	}
	fmt.Printf("Active coffee products (%d total):\n", page.TotalElements)
	for _, p := range page.Content {
		fmt.Printf("  %-12s  %d¢\n", p.Name, p.Price)
	}

	// Query 2: All active food products, sorted by price ascending.
	activeFood := And(ActiveProducts(), InCategory("food"))
	sorted, err := repo.Search(ctx,
		persistence.PageRequest{Page: 0, Size: 10},
		activeFood,
		persistence.Sort{Field: "price", Descending: false},
	)
	if err != nil {
		log.Fatal("Search:", err)
	}
	fmt.Printf("\nActive food products sorted by price (cheapest first):\n")
	for _, p := range sorted.Content {
		fmt.Printf("  %-12s  %d¢\n", p.Name, p.Price)
	}

	// Query 3: Pagination — 2 items per page.
	allActive := ActiveProducts()
	p0, _ := repo.FindAll(ctx, persistence.PageRequest{Page: 0, Size: 2}, allActive)
	p1, _ := repo.FindAll(ctx, persistence.PageRequest{Page: 1, Size: 2}, allActive)
	fmt.Printf("\nPagination (size=2): page 0 has %d items, page 1 has %d items, %d total pages\n",
		len(p0.Content), len(p1.Content), p0.TotalPages)

	// FindByID.
	found, ok, err := repo.FindByID(ctx, "p1")
	if err != nil || !ok {
		log.Fatal("FindByID p1:", err)
	}
	fmt.Printf("\nFound by ID: %s (%s)\n", found.Name, found.Category)

	// DeleteByID.
	if err := repo.DeleteByID(ctx, "p4"); err != nil {
		log.Fatal("DeleteByID:", err)
	}
	_, exists, _ := repo.FindByID(ctx, "p4")
	fmt.Printf("After delete: p4 exists = %v\n", exists)

	// Clear — useful between tests.
	repo.Clear()
	after, _ := repo.FindAll(ctx, persistence.PageRequest{Page: 0, Size: 100}, ActiveProducts())
	fmt.Printf("After Clear(): %d products remain\n", after.TotalElements)
}
