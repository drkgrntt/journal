-- The old per-feature purchase tables. Paid features are gated on the user's
-- subscription status now (see 000004_subscriptions) and nothing reads these.
-- Destructive: existing rows are discarded. Restore from a backup if needed.
DROP TABLE IF EXISTS public.user_features;
DROP TABLE IF EXISTS public.features;
DROP SEQUENCE IF EXISTS public.features_id_seq;
