ALTER TABLE public.users DROP COLUMN IF EXISTS email_confirmation_sent_at;
ALTER TABLE public.users DROP COLUMN IF EXISTS email_confirmed_at;
