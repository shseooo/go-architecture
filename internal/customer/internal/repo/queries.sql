-- name: CreateMember :execresult
INSERT INTO member (name, city, street, zipcode) VALUES (?, ?, ?, ?);

-- name: GetMember :one
SELECT id, name, city, street, zipcode FROM member WHERE id = ?;
