package app

import (
	"context"
	"strings"

	"github.com/shseooo/go-architecture/internal/customer/internal/domain"
	"github.com/shseooo/go-architecture/internal/shared"
)

// Repository is the persistence contract, defined by the consumer (this service).
type Repository interface {
	Create(ctx context.Context, m *domain.Member) error
	GetByID(ctx context.Context, id int64) (domain.Member, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, m *domain.Member) error {
	if strings.TrimSpace(m.Name) == "" {
		return shared.ErrBadParamInput
	}
	return s.repo.Create(ctx, m)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.Member, error) {
	return s.repo.GetByID(ctx, id)
}
