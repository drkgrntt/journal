-- Entry templates: user-authored, reusable starting text for a new journal
-- entry. name and body are encrypted at rest (is_encrypted), like reminders.text.

CREATE TABLE IF NOT EXISTS public.entry_templates (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT entry_templates_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    name text NOT NULL,
    body text NOT NULL,
    is_encrypted boolean NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_entry_templates_deleted_at ON public.entry_templates (deleted_at);
CREATE INDEX IF NOT EXISTS idx_entry_templates_creator_id ON public.entry_templates (creator_id) WHERE deleted_at IS NULL;
