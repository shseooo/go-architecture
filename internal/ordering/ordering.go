// Package ordering is the public API of the ordering bounded context. Because
// this context coordinates other contexts, its gateway contracts live here (so
// the composition root can satisfy them) and the orchestration lives on Module.
package ordering

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/shseooo/go-architecture/internal/ordering/internal/domain"
	"github.com/shseooo/go-architecture/internal/ordering/internal/repo"
	"github.com/shseooo/go-architecture/internal/ordering/internal/rest"
	"github.com/shseooo/go-architecture/internal/platform/database"
	"github.com/shseooo/go-architecture/internal/shared"
)

// Item is what the ordering context needs to know about a catalog item.
type Item struct {
	ID    int64
	Name  string
	Price int
}

// ItemGateway is ordering's view of the catalog context. Reserve/Restore must
// honor the ambient transaction (they run inside the order transaction).
type ItemGateway interface {
	Reserve(ctx context.Context, itemID int64, qty int) (Item, error)
	Restore(ctx context.Context, itemID int64, qty int) error
}

// MemberGateway is ordering's view of the customer context.
type MemberGateway interface {
	EnsureExists(ctx context.Context, memberID int64) error
}

// Module is the ordering context: it holds its own repo/tx and the gateways to
// the contexts it depends on, and it implements the HTTP service.
type Module struct {
	orders  *repo.OrderRepository
	tx      *database.TxManager
	items   ItemGateway
	members MemberGateway
	handler *rest.Handler
}

func New(db *sql.DB, items ItemGateway, members MemberGateway) *Module {
	m := &Module{
		orders:  repo.New(db),
		tx:      database.NewTxManager(db),
		items:   items,
		members: members,
	}
	m.handler = rest.New(m)
	return m
}

func (m *Module) Routes(mux *http.ServeMux) {
	m.handler.Routes(mux)
}

// Place creates an order: verify member, reserve stock for each line (in the
// catalog context), create the delivery and order — all in one transaction that
// spans both contexts because they share the database.
func (m *Module) Place(ctx context.Context, cmd domain.PlaceCommand) (domain.Order, error) {
	if len(cmd.Lines) == 0 {
		return domain.Order{}, shared.ErrBadParamInput
	}
	for _, l := range cmd.Lines {
		if l.Count <= 0 {
			return domain.Order{}, shared.ErrBadParamInput
		}
	}

	var placed domain.Order
	err := m.tx.Do(ctx, func(ctx context.Context) error {
		if err := m.members.EnsureExists(ctx, cmd.MemberID); err != nil {
			return err
		}

		orderItems := make([]domain.OrderItem, 0, len(cmd.Lines))
		for _, line := range cmd.Lines {
			item, err := m.items.Reserve(ctx, line.ItemID, line.Count)
			if err != nil {
				return err
			}
			orderItems = append(orderItems, domain.OrderItem{
				ItemID:     item.ID,
				ItemName:   item.Name,
				OrderPrice: item.Price,
				Count:      line.Count,
			})
		}

		delivery := &domain.Delivery{Status: domain.DeliveryStatusReady, Address: cmd.Address}
		if err := m.orders.CreateDelivery(ctx, delivery); err != nil {
			return err
		}

		order := &domain.Order{
			MemberID:   cmd.MemberID,
			DeliveryID: &delivery.ID,
			OrderDate:  time.Now(),
			Status:     domain.OrderStatusOrder,
			OrderItems: orderItems,
		}
		if err := m.orders.Create(ctx, order); err != nil {
			return err
		}
		order.Delivery = delivery
		placed = *order
		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return placed, nil
}

func (m *Module) FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error) {
	return m.orders.FindByMember(ctx, memberID)
}

// Cancel cancels an order and restores stock in the catalog context, atomically.
func (m *Module) Cancel(ctx context.Context, orderID int64) error {
	return m.tx.Do(ctx, func(ctx context.Context) error {
		order, err := m.orders.GetByID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := order.Cancel(); err != nil {
			return err
		}
		for _, oi := range order.OrderItems {
			if err := m.items.Restore(ctx, oi.ItemID, oi.Count); err != nil {
				return err
			}
		}
		return m.orders.UpdateStatus(ctx, order.ID, order.Status)
	})
}
