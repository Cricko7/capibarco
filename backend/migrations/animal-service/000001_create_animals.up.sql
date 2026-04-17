CREATE TABLE IF NOT EXISTS animals (
    animal_id TEXT PRIMARY KEY,
    owner_profile_id TEXT NOT NULL,
    owner_type INTEGER NOT NULL,
    name TEXT NOT NULL,
    species INTEGER NOT NULL,
    breed TEXT NOT NULL DEFAULT '',
    sex INTEGER NOT NULL,
    size INTEGER NOT NULL,
    age_months INTEGER,
    description TEXT NOT NULL DEFAULT '',
    traits JSONB NOT NULL DEFAULT '[]'::jsonb,
    medical_notes JSONB NOT NULL DEFAULT '[]'::jsonb,
    vaccinated BOOLEAN NOT NULL DEFAULT false,
    sterilized BOOLEAN NOT NULL DEFAULT false,
    status INTEGER NOT NULL,
    location JSONB NOT NULL DEFAULT '{}'::jsonb,
    photos JSONB NOT NULL DEFAULT '[]'::jsonb,
    visibility INTEGER NOT NULL,
    boosted BOOLEAN NOT NULL DEFAULT false,
    boost_expires_at TIMESTAMPTZ,
    audit_created_at TIMESTAMPTZ NOT NULL,
    audit_updated_at TIMESTAMPTZ NOT NULL,
    audit_created_by TEXT NOT NULL,
    audit_updated_by TEXT NOT NULL,
    donation_count BIGINT NOT NULL DEFAULT 0,
    interest_count BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT animals_age_months_non_negative CHECK (age_months IS NULL OR age_months >= 0),
    CONSTRAINT animals_owner_type_positive CHECK (owner_type > 0),
    CONSTRAINT animals_species_positive CHECK (species > 0),
    CONSTRAINT animals_sex_positive CHECK (sex > 0),
    CONSTRAINT animals_size_positive CHECK (size > 0)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    animal_id TEXT NOT NULL REFERENCES animals(animal_id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS animals_owner_profile_id_idx ON animals(owner_profile_id);
CREATE INDEX IF NOT EXISTS animals_status_visibility_idx ON animals(status, visibility);
CREATE INDEX IF NOT EXISTS animals_species_idx ON animals(species);
CREATE INDEX IF NOT EXISTS animals_boosted_idx ON animals(boosted, boost_expires_at);
CREATE INDEX IF NOT EXISTS animals_city_idx ON animals ((lower(location->>'city')));
CREATE INDEX IF NOT EXISTS animals_traits_gin_idx ON animals USING GIN (traits);
CREATE INDEX IF NOT EXISTS animals_updated_at_idx ON animals(audit_updated_at DESC, animal_id ASC);
