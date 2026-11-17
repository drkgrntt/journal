ALTER TABLE public.users DROP COLUMN IF EXISTS reminder_email_last_sent_on;
ALTER TABLE public.users DROP COLUMN IF EXISTS reminder_email_message;
ALTER TABLE public.users DROP COLUMN IF EXISTS reminder_email_time;
ALTER TABLE public.users DROP COLUMN IF EXISTS reminder_email_enabled;
ALTER TABLE public.users DROP COLUMN IF EXISTS timezone;
