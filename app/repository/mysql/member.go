package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shseooo/go-architecture/app/domain"
)

type MemberRepository struct {
	db *sql.DB
}

func NewMemberRepository(db *sql.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

func (r *MemberRepository) Create(ctx context.Context, m *domain.Member) error {
	const query = `INSERT INTO member (name, city, street, zipcode) VALUES (?, ?, ?, ?)`
	res, err := querier(ctx, r.db).ExecContext(ctx, query,
		m.Name, m.Address.City, m.Address.Street, m.Address.Zipcode)
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
	const query = `SELECT id, name, city, street, zipcode FROM member WHERE id = ?`
	var m domain.Member
	err := querier(ctx, r.db).QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.Name, &m.Address.City, &m.Address.Street, &m.Address.Zipcode)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Member{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Member{}, err
	}
	return m, nil
}
