CREATE TABLE IF NOT EXISTS paid_holidays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    holiday_date DATE NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, holiday_date)
);

CREATE INDEX IF NOT EXISTS idx_paid_holidays_household ON paid_holidays(household_id, holiday_date);
