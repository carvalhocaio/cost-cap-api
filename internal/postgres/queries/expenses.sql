-- name: CreateExpense :one
INSERT INTO expenses (id, user_id, description, amount_cents, category, spent_on)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetExpense :one
SELECT *
FROM expenses
WHERE id = $1 AND user_id = $2;

-- name: UpdateExpense :one
UPDATE expenses
SET description  = $3,
    amount_cents = $4,
    category     = $5,
    spent_on     = $6,
    updated_at   = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteExpense :execrows
DELETE FROM expenses
WHERE id = $1 AND user_id = $2;

-- name: ListExpenses :many
SELECT *
FROM expenses
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(from_date)::date IS NULL OR spent_on >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR spent_on <= sqlc.narg(to_date))
  AND (sqlc.narg(category)::expense_category IS NULL OR category = sqlc.narg(category))
  AND (
      sqlc.narg(cursor_spent_on)::date IS NULL
      OR (spent_on, id) < (sqlc.narg(cursor_spent_on), sqlc.narg(cursor_id)::uuid)
  )
ORDER BY spent_on DESC, id DESC
LIMIT sqlc.arg(page_size);
