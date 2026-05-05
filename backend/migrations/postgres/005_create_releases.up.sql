CREATE TABLE IF NOT EXISTS releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    git_tag TEXT NOT NULL DEFAULT '',
    branch TEXT NOT NULL DEFAULT '',
    environment TEXT NOT NULL CHECK (environment IN ('dev', 'staging', 'prod')),
    strategy TEXT NOT NULL CHECK (strategy IN ('rolling', 'recreate', 'canary')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'in_progress', 'success', 'failed')),
    description TEXT NOT NULL DEFAULT '',
    author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS releases_service_id_idx ON releases (service_id);
CREATE INDEX IF NOT EXISTS releases_author_user_id_idx ON releases (author_user_id);
CREATE INDEX IF NOT EXISTS releases_service_deployed_at_idx ON releases (service_id, deployed_at DESC);
