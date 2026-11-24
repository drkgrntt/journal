-- Server-side sessions. The cookie holds a random token; only its SHA-256 hash
-- is stored here, so a copy of the table can't be used to sign in. Rows hold no
-- IP address or user agent (see the anonymous request log).
-- Existing JWT cookies stop working when the app that reads this table ships,
-- so everyone signs in once more.
CREATE TABLE IF NOT EXISTS public.sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT sessions_pkey PRIMARY KEY,
    user_id uuid NOT NULL CONSTRAINT sessions_user_id_fkey REFERENCES public.users(id) ON DELETE CASCADE,
    token_hash text NOT NULL,
    previous_token_hash text,
    previous_valid_until timestamptz,
    created_at timestamptz NOT NULL,
    rotated_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash ON public.sessions USING btree (token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_previous_token_hash ON public.sessions USING btree (previous_token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON public.sessions USING btree (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON public.sessions USING btree (expires_at);
