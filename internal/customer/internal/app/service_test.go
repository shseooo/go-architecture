package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shseooo/go-architecture/internal/customer/internal/app"
	"github.com/shseooo/go-architecture/internal/customer/internal/domain"
	"github.com/shseooo/go-architecture/internal/shared"
)

type stubRepo struct {
	created *domain.Member
	getErr  error
}

func (s *stubRepo) Create(_ context.Context, m *domain.Member) error {
	m.ID = 42
	s.created = m
	return nil
}
func (s *stubRepo) GetByID(_ context.Context, id int64) (domain.Member, error) {
	if s.getErr != nil {
		return domain.Member{}, s.getErr
	}
	return domain.Member{ID: id, Name: "Alice"}, nil
}

func TestRegister_Success(t *testing.T) {
	repo := &stubRepo{}
	svc := app.NewService(repo)
	m := &domain.Member{Name: "Alice", Address: shared.Address{City: "Seoul"}}
	require.NoError(t, svc.Register(context.Background(), m))
	assert.Equal(t, int64(42), m.ID)
}

func TestRegister_EmptyName(t *testing.T) {
	svc := app.NewService(&stubRepo{})
	err := svc.Register(context.Background(), &domain.Member{Name: "  "})
	assert.ErrorIs(t, err, shared.ErrBadParamInput)
}

func TestGetByID_NotFound(t *testing.T) {
	svc := app.NewService(&stubRepo{getErr: shared.ErrNotFound})
	_, err := svc.GetByID(context.Background(), 1)
	assert.ErrorIs(t, err, shared.ErrNotFound)
}
