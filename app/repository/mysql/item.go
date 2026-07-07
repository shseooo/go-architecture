package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shseooo/go-architecture/app/domain"
)

type ItemRepository struct {
	db *sql.DB
}

func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// itemColumns is the shared select list for reading items.
const itemColumns = `id, name, price, stock_quantity, dtype, author, isbn, artist, etc, director, actor, created_at`

// rowScanner is implemented by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanItem reads one item row, mapping the nullable type-specific columns.
func scanItem(s rowScanner) (domain.Item, error) {
	var (
		it                                       domain.Item
		author, isbn, artist, etc, director, act sql.NullString
	)
	err := s.Scan(
		&it.ID, &it.Name, &it.Price, &it.StockQuantity, &it.Type,
		&author, &isbn, &artist, &etc, &director, &act, &it.CreatedAt,
	)
	if err != nil {
		return domain.Item{}, err
	}
	it.Author, it.ISBN = author.String, isbn.String
	it.Artist, it.Etc = artist.String, etc.String
	it.Director, it.Actor = director.String, act.String
	return it, nil
}

func (r *ItemRepository) Create(ctx context.Context, it *domain.Item) error {
	const query = `INSERT INTO item
		(name, price, stock_quantity, dtype, author, isbn, artist, etc, director, actor)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := querier(ctx, r.db).ExecContext(ctx, query,
		it.Name, it.Price, it.StockQuantity, it.Type,
		nullify(it.Author), nullify(it.ISBN), nullify(it.Artist), nullify(it.Etc),
		nullify(it.Director), nullify(it.Actor),
	)
	if err != nil {
		return err
	}
	if it.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return r.replaceCategories(ctx, it.ID, it.CategoryIDs)
}

func (r *ItemRepository) Update(ctx context.Context, it *domain.Item) error {
	const query = `UPDATE item SET
		name = ?, price = ?, stock_quantity = ?, dtype = ?,
		author = ?, isbn = ?, artist = ?, etc = ?, director = ?, actor = ?
		WHERE id = ?`
	res, err := querier(ctx, r.db).ExecContext(ctx, query,
		it.Name, it.Price, it.StockQuantity, it.Type,
		nullify(it.Author), nullify(it.ISBN), nullify(it.Artist), nullify(it.Etc),
		nullify(it.Director), nullify(it.Actor), it.ID,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return r.replaceCategories(ctx, it.ID, it.CategoryIDs)
}

func (r *ItemRepository) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	const query = `SELECT ` + itemColumns + ` FROM item WHERE id = ?`
	it, err := scanItem(querier(ctx, r.db).QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Item{}, err
	}
	if it.CategoryIDs, err = r.categoryIDs(ctx, id); err != nil {
		return domain.Item{}, err
	}
	return it, nil
}

// UpdateStock persists an absolute stock quantity. Used inside the order transaction.
func (r *ItemRepository) UpdateStock(ctx context.Context, itemID int64, quantity int) error {
	const query = `UPDATE item SET stock_quantity = ? WHERE id = ?`
	_, err := querier(ctx, r.db).ExecContext(ctx, query, quantity, itemID)
	return err
}

// itemFilterClause builds the shared JOIN + WHERE (and their args) for Search
// and Count. Values always go through placeholders — never string interpolation.
func itemFilterClause(f domain.ItemFilter) (join, where string, args []any) {
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

// Search runs the dynamic item query. The ORDER BY column comes from a closed
// allowlist (orderClause), so user input can never reach the SQL text. The
// filter's Limit/Offset are assumed normalized by the service.
func (r *ItemRepository) Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, error) {
	join, where, args := itemFilterClause(f)

	query := "SELECT item." + strings.ReplaceAll(itemColumns, ", ", ", item.") +
		" FROM item" + join + where +
		" ORDER BY " + orderClause(f.Sort) +
		" LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	rows, err := querier(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Item, 0)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Count returns the total number of items matching the filter (ignoring paging),
// used to build pagination metadata.
func (r *ItemRepository) Count(ctx context.Context, f domain.ItemFilter) (int, error) {
	join, where, args := itemFilterClause(f)
	query := "SELECT COUNT(*) FROM item" + join + where
	var total int
	err := querier(ctx, r.db).QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

// orderClause maps the closed ItemSort set to trusted SQL. Any unknown value
// falls back to a safe default — user input never becomes a column name.
func orderClause(s domain.ItemSort) string {
	switch s {
	case domain.ItemSortOldest:
		return "item.created_at ASC"
	case domain.ItemSortPriceAsc:
		return "item.price ASC"
	case domain.ItemSortPriceDesc:
		return "item.price DESC"
	case domain.ItemSortNewest:
		fallthrough
	default:
		return "item.created_at DESC"
	}
}

// replaceCategories resets the item's category links to the given set.
func (r *ItemRepository) replaceCategories(ctx context.Context, itemID int64, categoryIDs []int64) error {
	q := querier(ctx, r.db)
	if _, err := q.ExecContext(ctx, `DELETE FROM category_item WHERE item_id = ?`, itemID); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(categoryIDs))
	args := make([]any, 0, len(categoryIDs)*2)
	for i, cid := range categoryIDs {
		placeholders[i] = "(?, ?)"
		args = append(args, cid, itemID)
	}
	query := fmt.Sprintf(`INSERT INTO category_item (category_id, item_id) VALUES %s`,
		strings.Join(placeholders, ", "))
	_, err := q.ExecContext(ctx, query, args...)
	return err
}

func (r *ItemRepository) categoryIDs(ctx context.Context, itemID int64) ([]int64, error) {
	rows, err := querier(ctx, r.db).QueryContext(ctx,
		`SELECT category_id FROM category_item WHERE item_id = ?`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// nullify turns an empty string into a NULL so that unused type-specific columns
// stay NULL rather than empty strings.
func nullify(s string) any {
	if s == "" {
		return nil
	}
	return s
}
