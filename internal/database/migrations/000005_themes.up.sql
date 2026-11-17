-- Appearance preferences (paid "Themes" feature). Empty string = default.
ALTER TABLE users ADD COLUMN IF NOT EXISTS theme text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS font_family text NOT NULL DEFAULT '';
