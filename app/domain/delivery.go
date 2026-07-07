package domain

// DeliveryStatus is the shipping state of an order's delivery.
type DeliveryStatus string

const (
	DeliveryStatusReady DeliveryStatus = "READY" // not yet shipped
	DeliveryStatusComp  DeliveryStatus = "COMP"  // shipping completed
)

// Delivery holds the shipping information for an order.
type Delivery struct {
	ID      int64          `json:"id"`
	Status  DeliveryStatus `json:"status"`
	Address Address        `json:"address"`
}
