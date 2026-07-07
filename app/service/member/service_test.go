package member_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shseooo/go-architecture/app/domain"
	"github.com/shseooo/go-architecture/app/service/member"
)

type stubRepo struct {
	created *domain.Member
	getFn   func(int64) (domain.Member, error)
}

func (s *stubRepo) Create(_ context.Context, m *domain.Member) error {
	m.ID = 42
	s.created = m
	return nil
}
func (s *stubRepo) GetByID(_ context.Context, id int64) (domain.Member, error) {
	return s.getFn(id)
}

func TestRegister_Success(t *testing.T) {
	repo := &stubRepo{}
	svc := member.NewService(repo)

	m := &domain.Member{Name: "Alice", Address: domain.Address{City: "Seoul"}}
	err := svc.Register(context.Background(), m)

	require.NoError(t, err)
	assert.Equal(t, int64(42), m.ID)
	assert.Equal(t, "Alice", repo.created.Name)
}

func TestRegister_EmptyName(t *testing.T) {
	svc := member.NewService(&stubRepo{})
	err := svc.Register(context.Background(), &domain.Member{Name: "  "})
	assert.ErrorIs(t, err, domain.ErrBadParamInput)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &stubRepo{getFn: func(int64) (domain.Member, error) {
		return domain.Member{}, domain.ErrNotFound
	}}
	svc := member.NewService(repo)
	_, err := svc.GetByID(context.Background(), 1)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
