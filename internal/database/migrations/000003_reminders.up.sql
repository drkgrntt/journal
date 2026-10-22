-- Reminders: user-authored text (affirmations, verses, prayers) shown back on the
-- Today tab and new-entry page. No completion state, schedule, or journal link.

CREATE TABLE IF NOT EXISTS public.reminders (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT reminders_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    text text NOT NULL,
    is_encrypted boolean NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reminders_deleted_at ON public.reminders (deleted_at);
