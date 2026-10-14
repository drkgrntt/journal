-- A user can own a given feature once. The Stripe webhook can be delivered
-- more than once, so enforce it in the database and upsert in the handler.
-- If this fails on existing data, soft-delete the duplicate user_features rows first.

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_features_creator_feature ON user_features (creator_id, feature_id) WHERE deleted_at IS NULL;
