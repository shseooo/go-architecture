package mysql

import (
	"context"
	"database/sql"

	"github.com/shseooo/go-architecture/app/domain"
)

type DeliveryRepository struct {
	db *sql.DB
}

func NewDeliveryRepository(db *sql.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) Create(ctx context.Context, d *domain.Delivery) error {
	const query = `INSERT INTO delivery (status, city, street, zipcode) VALUES (?, ?, ?, ?)`
	res, err := querier(ctx, r.db).ExecContext(ctx, query,
		d.Status, d.Address.City, d.Address.Street, d.Address.Zipcode)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = id
	return nil
}
