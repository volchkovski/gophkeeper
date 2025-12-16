-- Create secrets table
CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    encrypted_data BYTEA NOT NULL,
    metadata TEXT,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_secrets_user_id_deleted_at ON secrets(user_id, deleted_at);

-- Unique constraint on user_id and name for non-deleted secrets
CREATE UNIQUE INDEX IF NOT EXISTS idx_secrets_user_id_name ON secrets(user_id, name) WHERE deleted_at IS NULL;

