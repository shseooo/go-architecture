package app

import (
	"context"
	"strings"

	"github.com/shseooo/go-architecture/internal/catalog/internal/domain"
	"github.com/shseooo/go-architecture/internal/shared"
)

type Repository interface {
	Create(ctx context.Context, it *domain.Item) error
	Update(ctx context.Context, it *domain.Item) error
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	UpdateStock(ctx context.Context, itemID int64, quantity int) error
	Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error)
	Count(ctx context.Context, f domain.ItemFilter) (int, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, it *domain.Item) error {
	if err := validate(it); err != nil {
		return err
	}
	return s.repo.Create(ctx, it)
}

func (s *Service) Update(ctx context.Context, it *domain.Item) error {
	if it.ID == 0 {
		return shared.ErrBadParamInput
	}
	if err := validate(it); err != nil {
		return err
	}
	return s.repo.Update(ctx, it)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	return s.repo.GetByID(ctx, id)
}

// Search returns a page plus the total count for pagination metadata.
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

// Reserve decrements stock for an ordered item. It runs within the caller's
// transaction (ordering places the whole order atomically).
func (s *Service) Reserve(ctx context.Context, itemID int64, qty int) (domain.Item, error) {
	it, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if err := it.RemoveStock(qty); err != nil {
		return domain.Item{}, err
	}
	if err := s.repo.UpdateStock(ctx, it.ID, it.StockQuantity); err != nil {
		return domain.Item{}, err
	}
	return it, nil
}

// Restore increments stock when an order is canceled.
func (s *Service) Restore(ctx context.Context, itemID int64, qty int) error {
	it, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	it.AddStock(qty)
	return s.repo.UpdateStock(ctx, it.ID, it.StockQuantity)
}

func validate(it *domain.Item) error {
	if strings.TrimSpace(it.Name) == "" || !it.Type.Valid() || it.Price < 0 || it.StockQuantity < 0 {
		return shared.ErrBadParamInput
	}
	return nil
}
