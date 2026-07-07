package member

import (
	"context"
	"strings"

	"github.com/shseooo/go-architecture/app/domain"
)

// Repository is the persistence contract the member service needs. It is defined
// here (by the consumer) so the service depends on an abstraction, not on MySQL.
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

// Register creates a new member. The member's ID is populated on success.
func (s *Service) Register(ctx context.Context, m *domain.Member) error {
	if strings.TrimSpace(m.Name) == "" {
		return domain.ErrBadParamInput
	}
	return s.repo.Create(ctx, m)
}

func (s *Service) GetByID(ctx context.Context, id int64) (domain.Member, error) {
	return s.repo.GetByID(ctx, id)
}
