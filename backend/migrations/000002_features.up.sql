ALTER TABLE transactions
    ALTER COLUMN paid_by_member_id DROP NOT NULL;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS is_joint BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS household_settings (
    household_id UUID PRIMARY KEY REFERENCES households(id) ON DELETE CASCADE,
    openai_api_key TEXT,
    openai_model TEXT NOT NULL DEFAULT 'gpt-4o',
    line_channel_secret TEXT,
    line_channel_access_token TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS line_identities (
    line_user_id TEXT PRIMARY KEY,
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    member_id UUID REFERENCES members(id) ON DELETE SET NULL,
    display_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS line_group_bindings (
    line_group_id TEXT PRIMARY KEY,
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pending_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_user_id TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pending_confirm_user ON pending_confirmations(line_user_id, expires_at);

CREATE TABLE IF NOT EXISTS caregivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    hourly_rate_satang BIGINT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS caregiver_hour_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    caregiver_id UUID NOT NULL REFERENCES caregivers(id) ON DELETE CASCADE,
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    work_date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    hours NUMERIC(6,2) NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_caregiver_hours_month ON caregiver_hour_entries(household_id, work_date);

CREATE TABLE IF NOT EXISTS lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pantry_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    expires_on DATE,
    notes TEXT NOT NULL DEFAULT '',
    rebuy_hint TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
