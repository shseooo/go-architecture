package item

import (
	"context"
	"strings"

	"github.com/shseooo/go-architecture/app/domain"
)

type Repository interface {
	Create(ctx context.Context, it *domain.Item) error
	Update(ctx context.Context, it *domain.Item) error
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error)
	Count(ctx context.Context, f domain.ItemFilter) (int, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new item after validating the invariants shared by every
// item type.
func (s *Service) Create(ctx context.Context, it *domain.Item) error {
	if err := validate(it); err != nil {
		return err
	}
	return s.repo.Create(ctx, it)
}

// Update modifies an existing item.
func (s *Service) Update(ctx context.Context, it *domain.Item) error {
	if it.ID == 0 {
		return domain.ErrBadParamInput
	}
	if err := validate(it); err != nil {
		return err
	}
	return s.repo.Update(ctx, it)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	return s.repo.GetByID(ctx, id)
}

// Search runs a dynamic query (category / price range / date sort) and returns
// the matching page together with the total count for pagination. The filter is
// normalized here so the query and the page metadata stay in sync.
func (s *Service) Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, int, error) {
	f.Normalize()
	items, err := s.repo.Search(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func validate(it *domain.Item) error {
	if strings.TrimSpace(it.Name) == "" {
		return domain.ErrBadParamInput
	}
	if !it.Type.Valid() {
		return domain.ErrBadParamInput
	}
	if it.Price < 0 || it.StockQuantity < 0 {
		return domain.ErrBadParamInput
	}
	return nil
}
