package item_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shseooo/go-architecture/app/domain"
	"github.com/shseooo/go-architecture/app/service/item"
)

type stubRepo struct {
	created  *domain.Item
	searchFn func(domain.ItemFilter) ([]domain.Item, error)
	total    int
}

func (s *stubRepo) Create(_ context.Context, it *domain.Item) error { s.created = it; return nil }
func (s *stubRepo) Update(context.Context, *domain.Item) error      { return nil }
func (s *stubRepo) GetByID(context.Context, int64) (domain.Item, error) {
	return domain.Item{}, nil
}
func (s *stubRepo) Search(_ context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	return s.searchFn(f)
}
func (s *stubRepo) Count(context.Context, domain.ItemFilter) (int, error) {
	return s.total, nil
}

func TestCreate_Success(t *testing.T) {
	repo := &stubRepo{}
	svc := item.NewService(repo)
	it := &domain.Item{Name: "Nevermind", Price: 100, StockQuantity: 5, Type: domain.ItemTypeAlbum, Artist: "Nirvana"}
	require.NoError(t, svc.Create(context.Background(), it))
	assert.Equal(t, "Nirvana", repo.created.Artist)
}

func TestCreate_InvalidType(t *testing.T) {
	svc := item.NewService(&stubRepo{})
	err := svc.Create(context.Background(), &domain.Item{Name: "X", Type: "GADGET"})
	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

func TestCreate_NegativePrice(t *testing.T) {
	svc := item.NewService(&stubRepo{})
	err := svc.Create(context.Background(), &domain.Item{Name: "X", Type: domain.ItemTypeBook, Price: -1})
	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

func TestSearch_PassesFilter(t *testing.T) {
	var got domain.ItemFilter
	repo := &stubRepo{total: 42, searchFn: func(f domain.ItemFilter) ([]domain.Item, error) {
		got = f
		return []domain.Item{{ID: 1}}, nil
	}}
	svc := item.NewService(repo)
	min := 10
	out, total, err := svc.Search(context.Background(), domain.ItemFilter{MinPrice: &min, Sort: domain.ItemSortPriceAsc})
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, 42, total)
	assert.Equal(t, 10, *got.MinPrice)
	assert.Equal(t, domain.ItemSortPriceAsc, got.Sort)
	assert.Equal(t, domain.DefaultItemLimit, got.Limit) // normalized
}
