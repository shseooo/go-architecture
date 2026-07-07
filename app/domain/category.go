package domain

// Category is a self-referencing hierarchy (parent_id points to another category).
// Items belong to categories through the many-to-many category_item join.
type Category struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id,omitempty"`
	Name     string `json:"name"`
}
