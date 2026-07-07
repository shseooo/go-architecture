package domain

import "time"

// ItemType is the discriminator for the single-table item inheritance (DTYPE column).
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

// Item is a product. It uses single-table inheritance: Type (DTYPE) selects which
// of the type-specific fields are meaningful (Book: Author/ISBN, Album: Artist/Etc,
// Movie: Director/Actor).
type Item struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Price         int       `json:"price"`
	StockQuantity int       `json:"stock_quantity"`
	Type          ItemType  `json:"type"`
	CreatedAt     time.Time `json:"created_at"`

	// Book
	Author string `json:"author,omitempty"`
	ISBN   string `json:"isbn,omitempty"`
	// Album
	Artist string `json:"artist,omitempty"`
	Etc    string `json:"etc,omitempty"`
	// Movie
	Director string `json:"director,omitempty"`
	Actor    string `json:"actor,omitempty"`

	// CategoryIDs the item belongs to (many-to-many via category_item).
	CategoryIDs []int64 `json:"category_ids,omitempty"`
}

// AddStock increases the stock quantity (used when an order is canceled).
func (i *Item) AddStock(quantity int) {
	i.StockQuantity += quantity
}

// RemoveStock decreases the stock quantity, returning ErrInsufficientStock if
// there is not enough on hand.
func (i *Item) RemoveStock(quantity int) error {
	if i.StockQuantity < quantity {
		return ErrInsufficientStock
	}
	i.StockQuantity -= quantity
	return nil
}

// ItemSort enumerates the allowed sort orders for item search. Keeping this a
// closed set is what makes dynamic ORDER BY safe against SQL injection.
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

// ItemFilter is the domain-level intent for a dynamic item search. The
// repository translates it into SQL; the service/handler never build SQL.
type ItemFilter struct {
	CategoryID *int64
	MinPrice   *int
	MaxPrice   *int
	Sort       ItemSort
	Limit      int
	Offset     int
}

// Normalize clamps the pagination window to a safe range so the same values are
// used for both the query and the response metadata.
func (f *ItemFilter) Normalize() {
	if f.Limit <= 0 || f.Limit > MaxItemLimit {
		f.Limit = DefaultItemLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
