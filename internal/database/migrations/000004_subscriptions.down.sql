DROP INDEX IF EXISTS public.idx_users_stripe_subscription_id;
DROP INDEX IF EXISTS public.idx_users_stripe_customer_id;
ALTER TABLE public.users DROP COLUMN IF EXISTS stripe_subscription_id;
ALTER TABLE public.users DROP COLUMN IF EXISTS current_period_end;
ALTER TABLE public.users DROP COLUMN IF EXISTS trial_ends_at;
ALTER TABLE public.users DROP COLUMN IF EXISTS subscription_status;
