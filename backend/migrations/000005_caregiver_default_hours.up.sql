ALTER TABLE caregivers
    ADD COLUMN IF NOT EXISTS default_start_time TIME,
    ADD COLUMN IF NOT EXISTS default_end_time TIME;
