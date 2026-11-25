-- Recreates the empty tables only; the dropped rows are not restored.
CREATE SEQUENCE IF NOT EXISTS public.features_id_seq START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
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
CREATE INDEX IF NOT EXISTS idx_features_deleted_at ON public.features USING btree (deleted_at);
CREATE INDEX IF NOT EXISTS idx_user_features_deleted_at ON public.user_features USING btree (deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_features_creator_feature ON public.user_features (creator_id, feature_id) WHERE deleted_at IS NULL;
