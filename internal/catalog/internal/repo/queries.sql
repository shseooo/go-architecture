-- name: CreateItem :execresult
INSERT INTO item (name, price, stock_quantity, dtype, author, isbn, artist, etc, director, actor)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateItem :execresult
UPDATE item SET
    name = ?, price = ?, stock_quantity = ?, dtype = ?,
    author = ?, isbn = ?, artist = ?, etc = ?, director = ?, actor = ?
WHERE id = ?;

-- name: GetItem :one
SELECT id, name, price, stock_quantity, dtype, author, isbn, artist, etc, director, actor, created_at
FROM item WHERE id = ?;

-- name: UpdateItemStock :exec
UPDATE item SET stock_quantity = ? WHERE id = ?;

-- name: DeleteItemCategories :exec
DELETE FROM category_item WHERE item_id = ?;

-- name: AddItemCategory :exec
INSERT INTO category_item (category_id, item_id) VALUES (?, ?);

-- name: ItemCategoryIDs :many
SELECT category_id FROM category_item WHERE item_id = ?;
