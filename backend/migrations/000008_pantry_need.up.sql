ALTER TABLE pantry_items
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'on_hand',
    ADD COLUMN IF NOT EXISTS needed_by DATE;

ALTER TABLE lists
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'custom';

CREATE UNIQUE INDEX IF NOT EXISTS idx_lists_one_shopping
    ON lists (household_id) WHERE kind = 'shopping';

ALTER TABLE list_items
    ADD COLUMN IF NOT EXISTS pantry_item_id UUID UNIQUE REFERENCES pantry_items(id) ON DELETE SET NULL;
