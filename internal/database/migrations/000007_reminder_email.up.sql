-- Reminder email: opt-in daily email (paid feature). The message is stored
-- encrypted; last_sent_on guarantees at most one send per local day.

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS timezone text NOT NULL DEFAULT '';
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS reminder_email_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS reminder_email_time text NOT NULL DEFAULT '08:00';
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS reminder_email_message text;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS reminder_email_last_sent_on date;
