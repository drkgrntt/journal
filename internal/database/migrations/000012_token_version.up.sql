-- Session and password-reset tokens carry this number. Bumping it (on a
-- password change or reset) ends every session and reset link issued before.
-- Existing sessions have no number and count as 0, so they stay valid.
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS token_version integer NOT NULL DEFAULT 0;
