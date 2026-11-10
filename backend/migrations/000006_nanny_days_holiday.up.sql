ALTER TABLE caregivers
    ADD COLUMN IF NOT EXISTS default_weekdays INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5,6}';

ALTER TABLE caregiver_hour_entries
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'extra';
