-- Email confirmation: reminder emails only go to a confirmed address.
-- Existing accounts are treated as confirmed. The backfill runs only when the
-- column is first added, so replaying this migration never confirms accounts
-- that signed up afterwards.

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'email_confirmed_at'
	) THEN
		ALTER TABLE public.users ADD COLUMN email_confirmed_at timestamptz;
		UPDATE public.users SET email_confirmed_at = now();
	END IF;
END $$;

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS email_confirmation_sent_at timestamptz;
