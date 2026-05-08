ALTER TABLE releases DROP CONSTRAINT IF EXISTS releases_strategy_check;
ALTER TABLE releases
    ADD CONSTRAINT releases_strategy_check
    CHECK (strategy IN ('rolling', 'recreate', 'canary'));
