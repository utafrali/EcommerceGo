DROP TABLE IF EXISTS campaign_stacking_rules;

ALTER TABLE campaigns DROP COLUMN IF EXISTS exclusion_group;
ALTER TABLE campaigns DROP COLUMN IF EXISTS priority;
ALTER TABLE campaigns DROP COLUMN IF EXISTS is_stackable;
