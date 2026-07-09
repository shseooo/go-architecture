package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shseooo/go-architecture/internal/catalog/internal/app"
	"github.com/shseooo/go-architecture/internal/catalog/internal/domain"
	"github.com/shseooo/go-architecture/internal/shared"
)

type stubRepo struct {
	items       map[int64]domain.Item
	stockWrites map[int64]int
	total       int
	searchFn    func(domain.ItemFilter) ([]domain.Item, error)
}

func (s *stubRepo) Create(_ context.Context, it *domain.Item) error { it.ID = 1; return nil }
func (s *stubRepo) Update(context.Context, *domain.Item) error      { return nil }
func (s *stubRepo) GetByID(_ context.Context, id int64) (domain.Item, error) {
	it, ok := s.items[id]
	if !ok {
		return domain.Item{}, shared.ErrNotFound
	}
	return it, nil
}
func (s *stubRepo) UpdateStock(_ context.Context, id int64, q int) error {
	if s.stockWrites == nil {
		s.stockWrites = map[int64]int{}
	}
	s.stockWrites[id] = q
	return nil
}
func (s *stubRepo) Search(_ context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	if s.searchFn != nil {
		return s.searchFn(f)
	}
	return nil, nil
}
func (s *stubRepo) Count(context.Context, domain.ItemFilter) (int, error) { return s.total, nil }

func TestCreate_InvalidType(t *testing.T) {
	svc := app.NewService(&stubRepo{})
	err := svc.Create(context.Background(), &domain.Item{Name: "X", Type: "GADGET"})
	assert.ErrorIs(t, err, shared.ErrBadParamInput)
}

func TestReserve_InsufficientStock(t *testing.T) {
	repo := &stubRepo{items: map[int64]domain.Item{1: {ID: 1, Price: 100, StockQuantity: 1}}}
	svc := app.NewService(repo)
	_, err := svc.Reserve(context.Background(), 1, 5)
	assert.ErrorIs(t, err, shared.ErrInsufficientStock)
	assert.Empty(t, repo.stockWrites) // no stock persisted on failure
}

func TestReserve_DecrementsStock(t *testing.T) {
	repo := &stubRepo{items: map[int64]domain.Item{1: {ID: 1, Name: "Book", Price: 100, StockQuantity: 10}}}
	svc := app.NewService(repo)
	it, err := svc.Reserve(context.Background(), 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 100, it.Price)
	assert.Equal(t, 7, repo.stockWrites[1]) // 10 - 3
}

func TestSearch_NormalizesAndReturnsTotal(t *testing.T) {
	var got domain.ItemFilter
	repo := &stubRepo{total: 42, searchFn: func(f domain.ItemFilter) ([]domain.Item, error) {
		got = f
		return []domain.Item{{ID: 1}}, nil
	}}
	svc := app.NewService(repo)
	items, total, err := svc.Search(context.Background(), domain.ItemFilter{Sort: domain.ItemSortPriceAsc})
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, 42, total)
	assert.Equal(t, domain.DefaultItemLimit, got.Limit) // normalized
}
