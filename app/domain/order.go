package domain

import "time"

// OrderStatus is the lifecycle state of an order.
type OrderStatus string

const (
	OrderStatusOrder  OrderStatus = "ORDER"
	OrderStatusCancel OrderStatus = "CANCEL"
)

// OrderItem is a single line of an order. ItemName is populated via JOIN on read
// (it is not stored on the order_item row).
type OrderItem struct {
	ID         int64  `json:"id"`
	OrderID    int64  `json:"order_id"`
	ItemID     int64  `json:"item_id"`
	ItemName   string `json:"item_name,omitempty"`
	OrderPrice int    `json:"order_price"`
	Count      int    `json:"count"`
}

// TotalPrice is the price for this line (unit order price × count).
func (oi OrderItem) TotalPrice() int {
	return oi.OrderPrice * oi.Count
}

// Order aggregates its order items and delivery. When loaded for a listing, the
// order items are fetched with a single JOIN to avoid the N+1 problem.
type Order struct {
	ID         int64       `json:"id"`
	MemberID   int64       `json:"member_id"`
	DeliveryID *int64      `json:"delivery_id,omitempty"`
	OrderDate  time.Time   `json:"order_date"`
	Status     OrderStatus `json:"status"`
	OrderItems []OrderItem `json:"order_items"`
	Delivery   *Delivery   `json:"delivery,omitempty"`
}

// TotalPrice sums the price of every order item.
func (o Order) TotalPrice() int {
	total := 0
	for _, oi := range o.OrderItems {
		total += oi.TotalPrice()
	}
	return total
}

// Cancel transitions the order to CANCEL. It returns ErrAlreadyCanceled if the
// order is not currently in ORDER status. Stock restoration is orchestrated by
// the service.
func (o *Order) Cancel() error {
	if o.Status != OrderStatusOrder {
		return ErrAlreadyCanceled
	}
	o.Status = OrderStatusCancel
	return nil
}
