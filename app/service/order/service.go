package order

import (
	"context"
	"time"

	"github.com/shseooo/go-architecture/app/domain"
)

// The order service orchestrates several repositories inside a single
// transaction (via TxManager), so placing or canceling an order is atomic:
// stock, delivery and order rows all commit together or not at all.

type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id int64) (domain.Order, error)
	FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error)
	UpdateStatus(ctx context.Context, orderID int64, status domain.OrderStatus) error
}

type ItemRepository interface {
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	UpdateStock(ctx context.Context, itemID int64, quantity int) error
}

type DeliveryRepository interface {
	Create(ctx context.Context, d *domain.Delivery) error
}

type MemberRepository interface {
	GetByID(ctx context.Context, id int64) (domain.Member, error)
}

// TxManager runs a function within a single database transaction.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	orders   OrderRepository
	items    ItemRepository
	delivery DeliveryRepository
	members  MemberRepository
	tx       TxManager
}

func NewService(
	orders OrderRepository,
	items ItemRepository,
	delivery DeliveryRepository,
	members MemberRepository,
	tx TxManager,
) *Service {
	return &Service{orders: orders, items: items, delivery: delivery, members: members, tx: tx}
}

// OrderLine is a requested item and quantity.
type OrderLine struct {
	ItemID int64
	Count  int
}

// PlaceOrderCommand is the input to placing an order.
type PlaceOrderCommand struct {
	MemberID int64
	Address  domain.Address
	Lines    []OrderLine
}

// Place creates an order: it checks and decrements stock for every line, creates
// the delivery, and stores the order with its items — all in one transaction.
func (s *Service) Place(ctx context.Context, cmd PlaceOrderCommand) (domain.Order, error) {
	if len(cmd.Lines) == 0 {
		return domain.Order{}, domain.ErrBadParamInput
	}
	for _, l := range cmd.Lines {
		if l.Count <= 0 {
			return domain.Order{}, domain.ErrBadParamInput
		}
	}

	var placed domain.Order
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		if _, err := s.members.GetByID(ctx, cmd.MemberID); err != nil {
			return err // ErrNotFound bubbles up if the member is unknown
		}

		orderItems := make([]domain.OrderItem, 0, len(cmd.Lines))
		for _, line := range cmd.Lines {
			item, err := s.items.GetByID(ctx, line.ItemID)
			if err != nil {
				return err
			}
			if err := item.RemoveStock(line.Count); err != nil {
				return err // ErrInsufficientStock
			}
			if err := s.items.UpdateStock(ctx, item.ID, item.StockQuantity); err != nil {
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
		if err := s.delivery.Create(ctx, delivery); err != nil {
			return err
		}

		order := &domain.Order{
			MemberID:   cmd.MemberID,
			DeliveryID: &delivery.ID,
			OrderDate:  time.Now(),
			Status:     domain.OrderStatusOrder,
			OrderItems: orderItems,
		}
		if err := s.orders.Create(ctx, order); err != nil {
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

// FindByMember returns a member's order history (N+1-safe at the repository).
func (s *Service) FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error) {
	return s.orders.FindByMember(ctx, memberID)
}

// Cancel cancels an order and restores the stock of every item, atomically.
func (s *Service) Cancel(ctx context.Context, orderID int64) error {
	return s.tx.Do(ctx, func(ctx context.Context) error {
		order, err := s.orders.GetByID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := order.Cancel(); err != nil {
			return err // ErrAlreadyCanceled
		}
		for _, oi := range order.OrderItems {
			item, err := s.items.GetByID(ctx, oi.ItemID)
			if err != nil {
				return err
			}
			item.AddStock(oi.Count)
			if err := s.items.UpdateStock(ctx, item.ID, item.StockQuantity); err != nil {
				return err
			}
		}
		return s.orders.UpdateStatus(ctx, order.ID, order.Status)
	})
}
