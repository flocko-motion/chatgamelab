-- Basic CRUD queries for core entities
-- Exactly four per table: create, read by id, update by id, delete by id.


-- game -----------------------------------------------------------------

-- name: GetGameByID :one
SELECT * FROM game WHERE id = $1;

