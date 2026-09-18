-- GORM's AutoMigrate created plain UNIQUE constraints on users.email and
-- custom_journal_types.name. Those constraints aren't scoped to exclude
-- soft-deleted rows, so once a user (or custom journal type) is soft-deleted
-- its email/name can never be reused. Replace each with a partial unique
-- index that only enforces uniqueness among non-deleted rows.

ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS uni_users_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;

ALTER TABLE IF EXISTS custom_journal_types DROP CONSTRAINT IF EXISTS uni_custom_journal_types_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_journal_types_name ON custom_journal_types (name) WHERE deleted_at IS NULL;
