-- name: CreateDelivery :execresult
INSERT INTO delivery (status, city, street, zipcode) VALUES (?, ?, ?, ?);

-- name: CreateOrder :execresult
INSERT INTO orders (member_id, delivery_id, order_date, status) VALUES (?, ?, ?, ?);

-- name: CreateOrderItem :execresult
INSERT INTO order_item (order_id, item_id, order_price, count) VALUES (?, ?, ?, ?);

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = ? WHERE id = ?;

-- name: GetOrder :one
SELECT id, member_id, delivery_id, order_date, status FROM orders WHERE id = ?;

-- name: ListOrdersByMember :many
SELECT id, member_id, delivery_id, order_date, status
FROM orders WHERE member_id = ? ORDER BY order_date DESC;

-- name: OrderItemsByOrderIDs :many
SELECT oi.id, oi.order_id, oi.item_id, i.name AS item_name, oi.order_price, oi.count
FROM order_item oi JOIN item i ON i.id = oi.item_id
WHERE oi.order_id IN (sqlc.slice('order_ids'));

-- name: DeliveriesByIDs :many
SELECT id, status, city, street, zipcode FROM delivery WHERE id IN (sqlc.slice('ids'));
