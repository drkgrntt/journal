-- Subscription state lives on the user. Replaces the one-time per-feature
-- purchases (features / user_features), which stay in place until the gating
-- change stops reading them.
--
-- subscription_status: '' (legacy, backfilled below), trialing, active,
-- past_due, canceled, lifetime, comped.

ALTER TABLE public.users ADD COLUMN IF NOT EXISTS subscription_status text NOT NULL DEFAULT '';
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS trial_ends_at timestamp with time zone;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS current_period_end timestamp with time zone;
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS stripe_subscription_id text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id ON public.users (stripe_customer_id) WHERE stripe_customer_id IS NOT NULL AND stripe_customer_id <> '';
CREATE INDEX IF NOT EXISTS idx_users_stripe_subscription_id ON public.users (stripe_subscription_id) WHERE stripe_subscription_id <> '';

-- Existing users get the standard 60-day trial starting now. Comp an account
-- afterwards with: UPDATE users SET subscription_status = 'comped' WHERE email = '...';
UPDATE public.users
SET subscription_status = 'trialing', trial_ends_at = now() + interval '60 days'
WHERE subscription_status = '';
