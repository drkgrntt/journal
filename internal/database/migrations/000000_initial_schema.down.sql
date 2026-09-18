-- Reverses 000000_initial_schema: drops every table this baseline creates.
-- Sequences owned by a dropped column (features/journal_types/ratings ids)
-- are dropped automatically along with their table. The uuid-ossp
-- extension is intentionally left in place rather than dropped.

DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.user_features;
DROP TABLE IF EXISTS public.thankfuls;
DROP TABLE IF EXISTS public.recurring_action_items;
DROP TABLE IF EXISTS public.ratings;
DROP TABLE IF EXISTS public.journals;
DROP TABLE IF EXISTS public.journal_types;
DROP TABLE IF EXISTS public.jobs;
DROP TABLE IF EXISTS public.features;
DROP TABLE IF EXISTS public.custom_journal_types;
DROP TABLE IF EXISTS public.analytics;
DROP TABLE IF EXISTS public.action_items;
