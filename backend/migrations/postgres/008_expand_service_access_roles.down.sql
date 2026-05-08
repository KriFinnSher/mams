ALTER TABLE service_access DROP CONSTRAINT IF EXISTS service_access_role_check;
ALTER TABLE service_access
    ADD CONSTRAINT service_access_role_check
    CHECK (role IN ('developer'));
