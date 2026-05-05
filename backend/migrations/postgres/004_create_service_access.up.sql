CREATE TABLE IF NOT EXISTS service_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('developer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT service_access_service_user_unique UNIQUE (service_id, user_id)
);

CREATE INDEX IF NOT EXISTS service_access_service_id_idx ON service_access (service_id);
CREATE INDEX IF NOT EXISTS service_access_user_id_idx ON service_access (user_id);
