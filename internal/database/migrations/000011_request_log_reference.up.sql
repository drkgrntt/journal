-- A random per-request reference, shown to the person on a server error so they
-- can quote it. It isn't derived from anything about them and is never shared
-- between requests. Rows are still deleted after 14 days.
ALTER TABLE public.analytics ADD COLUMN IF NOT EXISTS request_id text;
CREATE INDEX IF NOT EXISTS idx_analytics_request_id ON public.analytics USING btree (request_id);
