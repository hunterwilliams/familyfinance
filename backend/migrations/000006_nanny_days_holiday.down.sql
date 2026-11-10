ALTER TABLE caregiver_hour_entries
    DROP COLUMN IF EXISTS kind;

ALTER TABLE caregivers
    DROP COLUMN IF EXISTS default_weekdays;
