ALTER TABLE infractions ADD COLUMN IF NOT EXISTS punishment TEXT DEFAULT '';
ALTER TABLE infractions ADD COLUMN IF NOT EXISTS appeal_due TEXT;
ALTER TABLE infractions ADD COLUMN IF NOT EXISTS image_url TEXT;
ALTER TABLE infractions ADD COLUMN IF NOT EXISTS added_role TEXT;
ALTER TABLE infractions ADD COLUMN IF NOT EXISTS removed_role TEXT;

UPDATE infractions SET punishment = 'Severity: ' || severity || ', What: ' || what_punishment WHERE punishment = '';

ALTER TABLE infractions ALTER COLUMN punishment SET NOT NULL;

ALTER TABLE infractions DROP COLUMN IF EXISTS severity;
ALTER TABLE infractions DROP COLUMN IF EXISTS what_punishment;
ALTER TABLE infractions DROP COLUMN IF EXISTS till_when;
