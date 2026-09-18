-- Baseline schema, captured from the live database as of 2026-09-17 (after
-- migration 000001 and this session's model-tag fixes were already
-- applied there). This is the single source of truth for a fresh install
-- going forward: cmd/migrate no longer relies on GORM's AutoMigrate to
-- create the base schema. Every statement here is safe to run against an
-- already-existing database (IF NOT EXISTS / inline constraints skipped
-- as a whole when the table already exists), so this is also a safe no-op
-- against any database that was already built by AutoMigrate before this
-- migration system existed.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS public.action_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT action_items_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    text text NOT NULL,
    completed_at timestamp with time zone,
    journal_id uuid,
    recurring_action_item_id uuid,
    is_encrypted boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS public.analytics (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT analytics_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    method text,
    domain text,
    page text,
    query text,
    ip text,
    useragent text
);

CREATE TABLE IF NOT EXISTS public.custom_journal_types (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT custom_journal_types_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    name character varying(32) NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.features_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.features (
    id bigint NOT NULL DEFAULT nextval('public.features_id_seq') CONSTRAINT features_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    code character varying(32) NOT NULL CONSTRAINT uni_features_code UNIQUE,
    name character varying(32) NOT NULL CONSTRAINT uni_features_name UNIQUE,
    enabled_at timestamp with time zone,
    description text,
    stripe_price_id text
);

ALTER SEQUENCE public.features_id_seq OWNED BY public.features.id;

CREATE TABLE IF NOT EXISTS public.jobs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT jobs_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    type character varying(255),
    notes text,
    processed_at timestamp with time zone,
    scheduled_at timestamp with time zone,
    attempted_at timestamp with time zone,
    retries bigint DEFAULT 0,
    priority bigint DEFAULT 10
);

CREATE SEQUENCE IF NOT EXISTS public.journal_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.journal_types (
    id bigint NOT NULL DEFAULT nextval('public.journal_types_id_seq') CONSTRAINT journal_types_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    code character varying(32) NOT NULL CONSTRAINT uni_journal_types_code UNIQUE,
    name character varying(32) NOT NULL CONSTRAINT uni_journal_types_name UNIQUE
);

ALTER SEQUENCE public.journal_types_id_seq OWNED BY public.journal_types.id;

CREATE TABLE IF NOT EXISTS public.journals (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT journals_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    date timestamp with time zone NOT NULL,
    entry text NOT NULL,
    bookmarked_at timestamp with time zone,
    journal_type_id bigint NOT NULL,
    custom_journal_type_id uuid,
    rating_id bigint NOT NULL,
    is_encrypted boolean NOT NULL
);

CREATE SEQUENCE IF NOT EXISTS public.ratings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.ratings (
    id bigint NOT NULL DEFAULT nextval('public.ratings_id_seq') CONSTRAINT ratings_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    code character varying(32) NOT NULL CONSTRAINT uni_ratings_code UNIQUE,
    name character varying(32) NOT NULL CONSTRAINT uni_ratings_name UNIQUE,
    value bigint NOT NULL
);

ALTER SEQUENCE public.ratings_id_seq OWNED BY public.ratings.id;

CREATE TABLE IF NOT EXISTS public.recurring_action_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT recurring_action_items_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    text text NOT NULL,
    is_encrypted boolean NOT NULL,
    frequency bigint,
    starts_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.thankfuls (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT thankfuls_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    text text NOT NULL,
    journal_id uuid,
    is_encrypted boolean NOT NULL
);

CREATE TABLE IF NOT EXISTS public.user_features (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT user_features_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    feature_id bigint NOT NULL,
    enabled_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL CONSTRAINT users_pkey PRIMARY KEY,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    creator_id uuid NOT NULL,
    last_updater_id uuid NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb,
    email text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    password text NOT NULL,
    stripe_customer_id text
);

CREATE INDEX IF NOT EXISTS idx_action_items_deleted_at ON public.action_items USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_analytics_deleted_at ON public.analytics USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_custom_journal_types_deleted_at ON public.custom_journal_types USING btree (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_journal_types_name ON public.custom_journal_types USING btree (name) WHERE (deleted_at IS NULL);
CREATE INDEX IF NOT EXISTS idx_features_deleted_at ON public.features USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_jobs_deleted_at ON public.jobs USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_journal_types_deleted_at ON public.journal_types USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_journals_deleted_at ON public.journals USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_ratings_deleted_at ON public.ratings USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_recurring_action_items_deleted_at ON public.recurring_action_items USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_thankfuls_deleted_at ON public.thankfuls USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_features_deleted_at ON public.user_features USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON public.users USING btree (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON public.users USING btree (email) WHERE (deleted_at IS NULL);
