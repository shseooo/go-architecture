package domain

import (
	"time"

	"github.com/shseooo/go-architecture/internal/shared"
)

type OrderStatus string

const (
	OrderStatusOrder  OrderStatus = "ORDER"
	OrderStatusCancel OrderStatus = "CANCEL"
)

type DeliveryStatus string

const (
	DeliveryStatusReady DeliveryStatus = "READY"
	DeliveryStatusComp  DeliveryStatus = "COMP"
)

type Delivery struct {
	ID      int64
	Status  DeliveryStatus
	Address shared.Address
}

type OrderItem struct {
	ID         int64
	OrderID    int64
	ItemID     int64
	ItemName   string
	OrderPrice int
	Count      int
}

func (oi OrderItem) TotalPrice() int { return oi.OrderPrice * oi.Count }

type Order struct {
	ID         int64
	MemberID   int64
	DeliveryID *int64
	OrderDate  time.Time
	Status     OrderStatus
	OrderItems []OrderItem
	Delivery   *Delivery
}

func (o Order) TotalPrice() int {
	total := 0
	for _, oi := range o.OrderItems {
		total += oi.TotalPrice()
	}
	return total
}

// Cancel transitions to CANCEL, or returns ErrAlreadyCanceled.
func (o *Order) Cancel() error {
	if o.Status != OrderStatusOrder {
		return shared.ErrAlreadyCanceled
	}
	o.Status = OrderStatusCancel
	return nil
}

// Line is one requested item + quantity within a PlaceCommand.
type Line struct {
	ItemID int64
	Count  int
}

// PlaceCommand is the input to placing an order.
type PlaceCommand struct {
	MemberID int64
	Address  shared.Address
	Lines    []Line
}
