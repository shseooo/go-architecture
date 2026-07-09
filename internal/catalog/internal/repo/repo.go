// Package repo implements the catalog Repository. Static CRUD uses sqlc; the
// dynamic item search is hand-written SQL (sqlc cannot express variable WHERE),
// both joining any ambient transaction from the context.
package repo

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/shseooo/go-architecture/internal/catalog/internal/domain"
	"github.com/shseooo/go-architecture/internal/catalog/internal/repo/sqlcdb"
	"github.com/shseooo/go-architecture/internal/platform/database"
	"github.com/shseooo/go-architecture/internal/shared"
)

type ItemRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) q(ctx context.Context) *sqlcdb.Queries {
	if tx := database.TxFromContext(ctx); tx != nil {
		return sqlcdb.New(tx)
	}
	return sqlcdb.New(r.db)
}

// conn is the raw executor for the hand-written dynamic queries.
type conn interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *ItemRepository) conn(ctx context.Context) conn {
	if tx := database.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *ItemRepository) Create(ctx context.Context, it *domain.Item) error {
	res, err := r.q(ctx).CreateItem(ctx, sqlcdb.CreateItemParams{
		Name:          it.Name,
		Price:         int32(it.Price),
		StockQuantity: int32(it.StockQuantity),
		Dtype:         string(it.Type),
		Author:        ns(it.Author), Isbn: ns(it.ISBN),
		Artist: ns(it.Artist), Etc: ns(it.Etc),
		Director: ns(it.Director), Actor: ns(it.Actor),
	})
	if err != nil {
		return err
	}
	if it.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return r.replaceCategories(ctx, it.ID, it.CategoryIDs)
}

func (r *ItemRepository) Update(ctx context.Context, it *domain.Item) error {
	res, err := r.q(ctx).UpdateItem(ctx, sqlcdb.UpdateItemParams{
		Name:          it.Name,
		Price:         int32(it.Price),
		StockQuantity: int32(it.StockQuantity),
		Dtype:         string(it.Type),
		Author:        ns(it.Author), Isbn: ns(it.ISBN),
		Artist: ns(it.Artist), Etc: ns(it.Etc),
		Director: ns(it.Director), Actor: ns(it.Actor),
		ID: it.ID,
	})
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return shared.ErrNotFound
	}
	return r.replaceCategories(ctx, it.ID, it.CategoryIDs)
}

func (r *ItemRepository) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	row, err := r.q(ctx).GetItem(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, shared.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}
	it := toDomain(row)
	if it.CategoryIDs, err = r.q(ctx).ItemCategoryIDs(ctx, id); err != nil {
		return domain.Item{}, err
	}
	return it, nil
}

func (r *ItemRepository) UpdateStock(ctx context.Context, itemID int64, quantity int) error {
	return r.q(ctx).UpdateItemStock(ctx, sqlcdb.UpdateItemStockParams{
		StockQuantity: int32(quantity),
		ID:            itemID,
	})
}

func (r *ItemRepository) replaceCategories(ctx context.Context, itemID int64, categoryIDs []int64) error {
	q := r.q(ctx)
	if err := q.DeleteItemCategories(ctx, itemID); err != nil {
		return err
	}
	for _, cid := range categoryIDs {
		if err := q.AddItemCategory(ctx, sqlcdb.AddItemCategoryParams{CategoryID: cid, ItemID: itemID}); err != nil {
			return err
		}
	}
	return nil
}

// --- dynamic search (hand-written) -----------------------------------------

const itemCols = `id, name, price, stock_quantity, dtype, author, isbn, artist, etc, director, actor, created_at`

func filterClause(f domain.ItemFilter) (join, where string, args []any) {
	var conds []string
	if f.CategoryID != nil {
		join = ` JOIN category_item ci ON ci.item_id = item.id`
		conds = append(conds, "ci.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.MinPrice != nil {
		conds = append(conds, "item.price >= ?")
		args = append(args, *f.MinPrice)
	}
	if f.MaxPrice != nil {
		conds = append(conds, "item.price <= ?")
		args = append(args, *f.MaxPrice)
	}
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	return join, where, args
}

func (r *ItemRepository) Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	join, where, args := filterClause(f)
	query := "SELECT item." + strings.ReplaceAll(itemCols, ", ", ", item.") +
		" FROM item" + join + where +
		" ORDER BY " + orderClause(f.Sort) +
		" LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	rows, err := r.conn(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Item, 0)
	for rows.Next() {
		it, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *ItemRepository) Count(ctx context.Context, f domain.ItemFilter) (int, error) {
	join, where, args := filterClause(f)
	var total int
	err := r.conn(ctx).QueryRowContext(ctx, "SELECT COUNT(*) FROM item"+join+where, args...).Scan(&total)
	return total, err
}

// orderClause maps the closed ItemSort set to trusted SQL (injection-safe).
func orderClause(s domain.ItemSort) string {
	switch s {
	case domain.ItemSortOldest:
		return "item.created_at ASC"
	case domain.ItemSortPriceAsc:
		return "item.price ASC"
	case domain.ItemSortPriceDesc:
		return "item.price DESC"
	default:
		return "item.created_at DESC"
	}
}

// --- mapping ----------------------------------------------------------------

func toDomain(row sqlcdb.Item) domain.Item {
	return domain.Item{
		ID:            row.ID,
		Name:          row.Name,
		Price:         int(row.Price),
		StockQuantity: int(row.StockQuantity),
		Type:          domain.ItemType(row.Dtype),
		CreatedAt:     row.CreatedAt,
		Author:        row.Author.String, ISBN: row.Isbn.String,
		Artist: row.Artist.String, Etc: row.Etc.String,
		Director: row.Director.String, Actor: row.Actor.String,
	}
}

func scanRows(rows *sql.Rows) (domain.Item, error) {
	var (
		it                                       domain.Item
		author, isbn, artist, etc, director, act sql.NullString
	)
	err := rows.Scan(&it.ID, &it.Name, &it.Price, &it.StockQuantity, &it.Type,
		&author, &isbn, &artist, &etc, &director, &act, &it.CreatedAt)
	if err != nil {
		return domain.Item{}, err
	}
	it.Author, it.ISBN = author.String, isbn.String
	it.Artist, it.Etc = artist.String, etc.String
	it.Director, it.Actor = director.String, act.String
	return it, nil
}

func ns(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
