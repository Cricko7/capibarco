CREATE TABLE IF NOT EXISTS user_profiles (
  profile_id TEXT PRIMARY KEY,
  auth_user_id TEXT NOT NULL,
  profile_type SMALLINT NOT NULL,
  display_name TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  visibility SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_reviews (
  review_id TEXT PRIMARY KEY,
  target_profile_id TEXT NOT NULL REFERENCES user_profiles(profile_id) ON DELETE CASCADE,
  author_profile_id TEXT NOT NULL REFERENCES user_profiles(profile_id) ON DELETE CASCADE,
  rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  text TEXT NOT NULL,
  match_id TEXT,
  visibility SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_reviews_target_created_at ON user_reviews(target_profile_id, created_at DESC);
