-- name: schema_up
CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY,
    tickets    JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_updated_at ON users (updated_at);

-- name: schema_down
DROP INDEX IF EXISTS idx_users_updated_at;
DROP TABLE IF EXISTS users;
