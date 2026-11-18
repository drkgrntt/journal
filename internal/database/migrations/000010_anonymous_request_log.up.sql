-- The request log no longer records who made a request: no IP address, user
-- agent, or query string (and the middleware stops writing an account ID).
-- It records only when, which page, and the response status, and the jobs
-- package deletes rows after 14 days. Existing rows hold the old identifying
-- data, so they are deleted rather than kept.
DELETE FROM public.analytics;

ALTER TABLE public.analytics DROP COLUMN IF EXISTS query;
ALTER TABLE public.analytics DROP COLUMN IF EXISTS ip;
ALTER TABLE public.analytics DROP COLUMN IF EXISTS useragent;
ALTER TABLE public.analytics ADD COLUMN IF NOT EXISTS status integer;

-- The retention job deletes by age.
CREATE INDEX IF NOT EXISTS idx_analytics_created_at ON public.analytics USING btree (created_at);
