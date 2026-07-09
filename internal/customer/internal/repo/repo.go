// Package repo implements the customer service's Repository using sqlc-generated
// queries. It maps between the generated row types and the domain entities, and
// joins any ambient transaction from the context.
package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shseooo/go-architecture/internal/customer/internal/domain"
	"github.com/shseooo/go-architecture/internal/customer/internal/repo/sqlcdb"
	"github.com/shseooo/go-architecture/internal/platform/database"
	"github.com/shseooo/go-architecture/internal/shared"
)

type MemberRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// q returns sqlc queries bound to the ambient transaction if present, else the pool.
func (r *MemberRepository) q(ctx context.Context) *sqlcdb.Queries {
	if tx := database.TxFromContext(ctx); tx != nil {
		return sqlcdb.New(tx)
	}
	return sqlcdb.New(r.db)
}

func (r *MemberRepository) Create(ctx context.Context, m *domain.Member) error {
	res, err := r.q(ctx).CreateMember(ctx, sqlcdb.CreateMemberParams{
		Name:    m.Name,
		City:    ns(m.Address.City),
		Street:  ns(m.Address.Street),
		Zipcode: ns(m.Address.Zipcode),
	})
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

func (r *MemberRepository) GetByID(ctx context.Context, id int64) (domain.Member, error) {
	row, err := r.q(ctx).GetMember(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Member{}, shared.ErrNotFound
	}
	if err != nil {
		return domain.Member{}, err
	}
	return domain.Member{
		ID:   row.ID,
		Name: row.Name,
		Address: shared.Address{
			City:    row.City.String,
			Street:  row.Street.String,
			Zipcode: row.Zipcode.String,
		},
	}, nil
}

// ns converts an empty string to NULL so unused columns stay NULL.
func ns(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
