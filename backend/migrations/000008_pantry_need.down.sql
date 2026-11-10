ALTER TABLE list_items DROP COLUMN IF EXISTS pantry_item_id;
DROP INDEX IF EXISTS idx_lists_one_shopping;
ALTER TABLE lists DROP COLUMN IF EXISTS kind;
ALTER TABLE pantry_items DROP COLUMN IF EXISTS needed_by;
ALTER TABLE pantry_items DROP COLUMN IF EXISTS status;
