// Package repo implements order persistence with sqlc, joining any ambient
// transaction. Order listing batch-loads items and deliveries (no N+1).
package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shseooo/go-architecture/internal/ordering/internal/domain"
	"github.com/shseooo/go-architecture/internal/ordering/internal/repo/sqlcdb"
	"github.com/shseooo/go-architecture/internal/platform/database"
	"github.com/shseooo/go-architecture/internal/shared"
)

type OrderRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) q(ctx context.Context) *sqlcdb.Queries {
	if tx := database.TxFromContext(ctx); tx != nil {
		return sqlcdb.New(tx)
	}
	return sqlcdb.New(r.db)
}

func (r *OrderRepository) CreateDelivery(ctx context.Context, d *domain.Delivery) error {
	res, err := r.q(ctx).CreateDelivery(ctx, sqlcdb.CreateDeliveryParams{
		Status:  string(d.Status),
		City:    ns(d.Address.City),
		Street:  ns(d.Address.Street),
		Zipcode: ns(d.Address.Zipcode),
	})
	if err != nil {
		return err
	}
	d.ID, err = res.LastInsertId()
	return err
}

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	q := r.q(ctx)
	res, err := q.CreateOrder(ctx, sqlcdb.CreateOrderParams{
		MemberID:   o.MemberID,
		DeliveryID: ni(o.DeliveryID),
		OrderDate:  o.OrderDate,
		Status:     string(o.Status),
	})
	if err != nil {
		return err
	}
	if o.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	for i := range o.OrderItems {
		oi := &o.OrderItems[i]
		oi.OrderID = o.ID
		res, err := q.CreateOrderItem(ctx, sqlcdb.CreateOrderItemParams{
			OrderID:    o.ID,
			ItemID:     oi.ItemID,
			OrderPrice: int32(oi.OrderPrice),
			Count:      int32(oi.Count),
		})
		if err != nil {
			return err
		}
		if oi.ID, err = res.LastInsertId(); err != nil {
			return err
		}
	}
	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus) error {
	return r.q(ctx).UpdateOrderStatus(ctx, sqlcdb.UpdateOrderStatusParams{
		Status: string(status),
		ID:     orderID,
	})
}

func (r *OrderRepository) GetByID(ctx context.Context, id int64) (domain.Order, error) {
	row, err := r.q(ctx).GetOrder(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, shared.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	o := toOrder(row)

	items, err := r.itemsByOrders(ctx, []int64{o.ID})
	if err != nil {
		return domain.Order{}, err
	}
	o.OrderItems = items[o.ID]

	if o.DeliveryID != nil {
		ds, err := r.deliveriesByIDs(ctx, []int64{*o.DeliveryID})
		if err != nil {
			return domain.Order{}, err
		}
		if d, ok := ds[*o.DeliveryID]; ok {
			o.Delivery = &d
		}
	}
	return o, nil
}

func (r *OrderRepository) FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error) {
	rows, err := r.q(ctx).ListOrdersByMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(rows))
	var orderIDs, deliveryIDs []int64
	for _, row := range rows {
		o := toOrder(row)
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
		if o.DeliveryID != nil {
			deliveryIDs = append(deliveryIDs, *o.DeliveryID)
		}
	}
	if len(orders) == 0 {
		return orders, nil
	}

	items, err := r.itemsByOrders(ctx, orderIDs) // 1 query
	if err != nil {
		return nil, err
	}
	deliveries, err := r.deliveriesByIDs(ctx, deliveryIDs) // 1 query
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].OrderItems = items[orders[i].ID]
		if orders[i].DeliveryID != nil {
			if d, ok := deliveries[*orders[i].DeliveryID]; ok {
				orders[i].Delivery = &d
			}
		}
	}
	return orders, nil
}

func (r *OrderRepository) itemsByOrders(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderItem, error) {
	out := make(map[int64][]domain.OrderItem)
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := r.q(ctx).OrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.OrderID] = append(out[row.OrderID], domain.OrderItem{
			ID:         row.ID,
			OrderID:    row.OrderID,
			ItemID:     row.ItemID,
			ItemName:   row.ItemName,
			OrderPrice: int(row.OrderPrice),
			Count:      int(row.Count),
		})
	}
	return out, nil
}

func (r *OrderRepository) deliveriesByIDs(ctx context.Context, ids []int64) (map[int64]domain.Delivery, error) {
	out := make(map[int64]domain.Delivery)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.q(ctx).DeliveriesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = domain.Delivery{
			ID:     row.ID,
			Status: domain.DeliveryStatus(row.Status),
			Address: shared.Address{
				City: row.City.String, Street: row.Street.String, Zipcode: row.Zipcode.String,
			},
		}
	}
	return out, nil
}

func toOrder(row sqlcdb.Order) domain.Order {
	o := domain.Order{
		ID:        row.ID,
		MemberID:  row.MemberID,
		OrderDate: row.OrderDate,
		Status:    domain.OrderStatus(row.Status),
	}
	if row.DeliveryID.Valid {
		o.DeliveryID = &row.DeliveryID.Int64
	}
	return o
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func ni(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
