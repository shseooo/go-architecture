package domain

import (
	"time"

	"github.com/shseooo/go-architecture/internal/shared"
)

// ItemType is the discriminator for single-table item inheritance (dtype column).
type ItemType string

const (
	ItemTypeBook  ItemType = "BOOK"
	ItemTypeAlbum ItemType = "ALBUM"
	ItemTypeMovie ItemType = "MOVIE"
)

func (t ItemType) Valid() bool {
	switch t {
	case ItemTypeBook, ItemTypeAlbum, ItemTypeMovie:
		return true
	default:
		return false
	}
}

// Item is a product. Type selects which type-specific fields are meaningful.
type Item struct {
	ID            int64
	Name          string
	Price         int
	StockQuantity int
	Type          ItemType
	CreatedAt     time.Time

	Author   string // book
	ISBN     string // book
	Artist   string // album
	Etc      string // album
	Director string // movie
	Actor    string // movie

	CategoryIDs []int64
}

// AddStock increases stock (order canceled).
func (i *Item) AddStock(qty int) { i.StockQuantity += qty }

// RemoveStock decreases stock, or returns ErrInsufficientStock.
func (i *Item) RemoveStock(qty int) error {
	if i.StockQuantity < qty {
		return shared.ErrInsufficientStock
	}
	i.StockQuantity -= qty
	return nil
}

// ItemSort is the closed set of allowed sort orders (safe dynamic ORDER BY).
type ItemSort string

const (
	ItemSortNewest    ItemSort = "newest"
	ItemSortOldest    ItemSort = "oldest"
	ItemSortPriceAsc  ItemSort = "price_asc"
	ItemSortPriceDesc ItemSort = "price_desc"
)

// Pagination defaults for item search.
const (
	DefaultItemLimit = 20
	MaxItemLimit     = 100
)

// ItemFilter is the domain-level intent for a dynamic item search.
type ItemFilter struct {
	CategoryID *int64
	MinPrice   *int
	MaxPrice   *int
	Sort       ItemSort
	Limit      int
	Offset     int
}

// Normalize clamps the pagination window so query and metadata stay in sync.
func (f *ItemFilter) Normalize() {
	if f.Limit <= 0 || f.Limit > MaxItemLimit {
		f.Limit = DefaultItemLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
