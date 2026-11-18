DROP INDEX IF EXISTS public.idx_analytics_request_id;
ALTER TABLE public.analytics DROP COLUMN IF EXISTS request_id;
