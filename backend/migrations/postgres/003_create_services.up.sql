CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    owner_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL CHECK (type IN ('business', 'composition')),
    version TEXT NOT NULL DEFAULT '',
    test_coverage INTEGER NOT NULL DEFAULT 0 CHECK (test_coverage >= 0 AND test_coverage <= 100),
    minimum_test_coverage_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    minimum_test_coverage INTEGER NOT NULL DEFAULT 0 CHECK (minimum_test_coverage >= 0 AND minimum_test_coverage <= 100),
    pii_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    responsible_team_ref TEXT NOT NULL DEFAULT '',
    importance TEXT NOT NULL CHECK (importance IN ('low', 'medium', 'high', 'critical')),
    repository_url TEXT NOT NULL,
    default_branch TEXT NOT NULL,
    grafana_dashboard_uid TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT services_organization_name_unique UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS services_organization_id_idx ON services (organization_id);
CREATE INDEX IF NOT EXISTS services_owner_user_id_idx ON services (owner_user_id);
