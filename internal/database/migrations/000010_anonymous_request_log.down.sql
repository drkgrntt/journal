-- Restores the columns only; the deleted rows and their data are not recoverable.
DROP INDEX IF EXISTS public.idx_analytics_created_at;
ALTER TABLE public.analytics DROP COLUMN IF EXISTS status;
ALTER TABLE public.analytics ADD COLUMN IF NOT EXISTS query text;
ALTER TABLE public.analytics ADD COLUMN IF NOT EXISTS ip text;
ALTER TABLE public.analytics ADD COLUMN IF NOT EXISTS useragent text;
