package domain

// Address is a value object embedded into Member and Delivery.
// It is flattened into city/street/zipcode columns at the persistence layer.
type Address struct {
	City    string `json:"city"`
	Street  string `json:"street"`
	Zipcode string `json:"zipcode"`
}

// Member represents a customer.
type Member struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	Address Address `json:"address"`
}
