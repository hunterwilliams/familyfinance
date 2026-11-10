ALTER TABLE caregivers
    DROP COLUMN IF EXISTS default_start_time,
    DROP COLUMN IF EXISTS default_end_time;
