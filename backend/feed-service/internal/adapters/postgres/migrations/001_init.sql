CREATE TABLE IF NOT EXISTS feed_schema_migrations (
	version integer PRIMARY KEY,
	name text NOT NULL,
	applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS feed_candidates (
	animal_id text PRIMARY KEY,
	animal_proto bytea NOT NULL,
	owner_display_name text NOT NULL DEFAULT '',
	owner_average_rating double precision NOT NULL DEFAULT 0,
	ranking_reasons text[] NOT NULL DEFAULT '{}',
	score_components jsonb NOT NULL DEFAULT '{}',
	distance_km integer NOT NULL DEFAULT 0,
	owner_hidden boolean NOT NULL DEFAULT false,
	owner_blocked boolean NOT NULL DEFAULT false,
	species integer NOT NULL DEFAULT 0,
	status integer NOT NULL DEFAULT 0,
	city text NOT NULL DEFAULT '',
	boosted boolean NOT NULL DEFAULT false,
	owner_profile_id text NOT NULL DEFAULT '',
	vaccinated boolean NOT NULL DEFAULT false,
	sterilized boolean NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS feed_candidates_filter_idx
	ON feed_candidates (status, species, city, boosted, owner_profile_id);

CREATE TABLE IF NOT EXISTS feed_served_cards (
	feed_card_id text PRIMARY KEY,
	animal_id text NOT NULL,
	feed_card_proto bytea NOT NULL,
	score_components jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS feed_served_cards_animal_idx
	ON feed_served_cards (animal_id);

CREATE TABLE IF NOT EXISTS feed_card_opens (
	idempotency_key text PRIMARY KEY,
	card_open_id text NOT NULL,
	opened_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS feed_seen_animals (
	actor_id text NOT NULL,
	animal_id text NOT NULL,
	PRIMARY KEY (actor_id, animal_id)
);

CREATE TABLE IF NOT EXISTS feed_advanced_filter_entitlements (
	owner_profile_id text PRIMARY KEY
);
