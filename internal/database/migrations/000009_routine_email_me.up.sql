-- Routines can optionally email the owner whenever one is created.
ALTER TABLE public.recurring_action_items ADD COLUMN IF NOT EXISTS email_me boolean NOT NULL DEFAULT false;
