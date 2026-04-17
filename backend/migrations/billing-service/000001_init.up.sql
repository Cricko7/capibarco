CREATE TABLE IF NOT EXISTS donations (
    donation_id TEXT PRIMARY KEY,
    payer_profile_id TEXT NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('shelter', 'animal')),
    target_id TEXT NOT NULL,
    currency_code CHAR(3) NOT NULL,
    units BIGINT NOT NULL CHECK (units >= 0),
    nanos INTEGER NOT NULL CHECK (nanos >= 0 AND nanos < 1000000000),
    status TEXT NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed', 'cancelled', 'refunded')),
    provider TEXT NOT NULL,
    provider_payment_id TEXT NOT NULL DEFAULT '',
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (units > 0 OR nanos > 0)
);

CREATE INDEX IF NOT EXISTS idx_donations_profile_created ON donations (payer_profile_id, created_at DESC, donation_id DESC);
CREATE INDEX IF NOT EXISTS idx_donations_target ON donations (target_type, target_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_provider_payment ON donations (provider, provider_payment_id) WHERE provider_payment_id <> '';

CREATE TABLE IF NOT EXISTS boosts (
    boost_id TEXT PRIMARY KEY,
    animal_id TEXT NOT NULL,
    owner_profile_id TEXT NOT NULL,
    donation_id TEXT NOT NULL REFERENCES donations(donation_id),
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    cancel_reason TEXT,
    cancelled_at TIMESTAMPTZ,
    CHECK (expires_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_boosts_animal_active ON boosts (animal_id, active, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_boosts_owner ON boosts (owner_profile_id, expires_at DESC);

CREATE TABLE IF NOT EXISTS entitlements (
    entitlement_id TEXT PRIMARY KEY,
    owner_profile_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('advanced_filters', 'extended_animal_stats', 'animal_boost')),
    resource_id TEXT,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    CHECK (expires_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_entitlements_owner_active ON entitlements (owner_profile_id, active, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_entitlements_resource ON entitlements (resource_id) WHERE resource_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS ledger_entries (
    ledger_entry_id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    currency_code CHAR(3) NOT NULL,
    units BIGINT NOT NULL CHECK (units >= 0),
    nanos INTEGER NOT NULL CHECK (nanos >= 0 AND nanos < 1000000000),
    reason TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CHECK (units > 0 OR nanos > 0)
);

CREATE INDEX IF NOT EXISTS idx_ledger_profile_created ON ledger_entries (profile_id, created_at DESC, ledger_entry_id DESC);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    scope TEXT NOT NULL,
    key_hash CHAR(64) NOT NULL,
    resource_kind TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    related_resource_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (scope, key_hash)
);

CREATE TABLE IF NOT EXISTS archived_animals (
    animal_id TEXT PRIMARY KEY,
    owner_profile_id TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS customers (
    profile_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL DEFAULT '',
    roles TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
