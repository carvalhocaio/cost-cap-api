-- +goose Up
CREATE TYPE expense_category AS ENUM (
    'groceries',
    'leisure',
    'electronics',
    'utilities',
    'clothing',
    'health',
    'others'
);

CREATE TABLE expenses (
    id           uuid             PRIMARY KEY,
    user_id      uuid             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    description  text             NOT NULL CHECK (char_length(description) BETWEEN 1 AND 255),
    amount_cents bigint           NOT NULL CHECK (amount_cents > 0),
    category     expense_category NOT NULL,
    spent_on     date             NOT NULL,
    created_at   timestamptz      NOT NULL DEFAULT now(),
    updated_at   timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX expenses_user_keyset_idx ON expenses (user_id, spent_on DESC, id DESC);

-- +goose Down
DROP TABLE expenses;
DROP TYPE expense_category;
