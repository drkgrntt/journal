DROP INDEX IF EXISTS idx_custom_journal_types_name;
ALTER TABLE custom_journal_types ADD CONSTRAINT uni_custom_journal_types_name UNIQUE (name);

DROP INDEX IF EXISTS idx_users_email;
ALTER TABLE users ADD CONSTRAINT uni_users_email UNIQUE (email);
