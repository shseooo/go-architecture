package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shseooo/go-architecture/app/domain"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts the order row and all of its order items. It expects to run
// inside a transaction (see TxManager) alongside the stock updates and delivery
// insert that make up placing an order.
func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	q := querier(ctx, r.db)
	const orderQuery = `INSERT INTO orders (member_id, delivery_id, order_date, status) VALUES (?, ?, ?, ?)`
	res, err := q.ExecContext(ctx, orderQuery, o.MemberID, o.DeliveryID, o.OrderDate, o.Status)
	if err != nil {
		return err
	}
	if o.ID, err = res.LastInsertId(); err != nil {
		return err
	}

	if len(o.OrderItems) == 0 {
		return nil
	}
	placeholders := make([]string, len(o.OrderItems))
	args := make([]any, 0, len(o.OrderItems)*4)
	for i := range o.OrderItems {
		placeholders[i] = "(?, ?, ?, ?)"
		oi := &o.OrderItems[i]
		oi.OrderID = o.ID
		args = append(args, oi.OrderID, oi.ItemID, oi.OrderPrice, oi.Count)
	}
	query := fmt.Sprintf(
		`INSERT INTO order_item (order_id, item_id, order_price, count) VALUES %s`,
		strings.Join(placeholders, ", "))
	_, err = q.ExecContext(ctx, query, args...)
	return err
}

// UpdateStatus changes an order's status (used to cancel).
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus) error {
	const query = `UPDATE orders SET status = ? WHERE id = ?`
	_, err := querier(ctx, r.db).ExecContext(ctx, query, status, orderID)
	return err
}

func (r *OrderRepository) GetByID(ctx context.Context, id int64) (domain.Order, error) {
	const query = `SELECT id, member_id, delivery_id, order_date, status FROM orders WHERE id = ?`
	o, err := scanOrder(querier(ctx, r.db).QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}

	items, err := r.orderItemsByOrderIDs(ctx, []int64{o.ID})
	if err != nil {
		return domain.Order{}, err
	}
	o.OrderItems = items[o.ID]

	if o.DeliveryID != nil {
		deliveries, err := r.deliveriesByIDs(ctx, []int64{*o.DeliveryID})
		if err != nil {
			return domain.Order{}, err
		}
		if d, ok := deliveries[*o.DeliveryID]; ok {
			o.Delivery = &d
		}
	}
	return o, nil
}

// FindByMember returns a member's orders. It loads order items and deliveries in
// bounded batches (one query each via IN clauses), so the query count does not
// grow with the number of orders — no N+1.
func (r *OrderRepository) FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error) {
	const query = `SELECT id, member_id, delivery_id, order_date, status
		FROM orders WHERE member_id = ? ORDER BY order_date DESC`
	rows, err := querier(ctx, r.db).QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	var orderIDs, deliveryIDs []int64
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
		if o.DeliveryID != nil {
			deliveryIDs = append(deliveryIDs, *o.DeliveryID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return orders, nil
	}

	itemsByOrder, err := r.orderItemsByOrderIDs(ctx, orderIDs) // 1 query
	if err != nil {
		return nil, err
	}
	deliveries, err := r.deliveriesByIDs(ctx, deliveryIDs) // 1 query
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].OrderItems = itemsByOrder[orders[i].ID]
		if orders[i].DeliveryID != nil {
			if d, ok := deliveries[*orders[i].DeliveryID]; ok {
				orders[i].Delivery = &d
			}
		}
	}
	return orders, nil
}

func scanOrder(s rowScanner) (domain.Order, error) {
	var (
		o          domain.Order
		deliveryID sql.NullInt64
	)
	err := s.Scan(&o.ID, &o.MemberID, &deliveryID, &o.OrderDate, &o.Status)
	if err != nil {
		return domain.Order{}, err
	}
	if deliveryID.Valid {
		o.DeliveryID = &deliveryID.Int64
	}
	return o, nil
}

// orderItemsByOrderIDs loads all order items for the given orders in a single
// query, joined with item for the display name, grouped by order id.
func (r *OrderRepository) orderItemsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderItem, error) {
	result := make(map[int64][]domain.OrderItem)
	if len(orderIDs) == 0 {
		return result, nil
	}
	ph, args := inClause(orderIDs)
	query := `SELECT oi.id, oi.order_id, oi.item_id, i.name, oi.order_price, oi.count
		FROM order_item oi JOIN item i ON i.id = oi.item_id
		WHERE oi.order_id IN (` + ph + `)`
	rows, err := querier(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var oi domain.OrderItem
		if err := rows.Scan(&oi.ID, &oi.OrderID, &oi.ItemID, &oi.ItemName, &oi.OrderPrice, &oi.Count); err != nil {
			return nil, err
		}
		result[oi.OrderID] = append(result[oi.OrderID], oi)
	}
	return result, rows.Err()
}

func (r *OrderRepository) deliveriesByIDs(ctx context.Context, ids []int64) (map[int64]domain.Delivery, error) {
	result := make(map[int64]domain.Delivery)
	if len(ids) == 0 {
		return result, nil
	}
	ph, args := inClause(ids)
	query := `SELECT id, status, city, street, zipcode FROM delivery WHERE id IN (` + ph + `)`
	rows, err := querier(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d domain.Delivery
		if err := rows.Scan(&d.ID, &d.Status, &d.Address.City, &d.Address.Street, &d.Address.Zipcode); err != nil {
			return nil, err
		}
		result[d.ID] = d
	}
	return result, rows.Err()
}

// inClause builds "?, ?, ..." and the matching args for an IN expression.
func inClause(ids []int64) (string, []any) {
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	return strings.Join(ph, ", "), args
}
