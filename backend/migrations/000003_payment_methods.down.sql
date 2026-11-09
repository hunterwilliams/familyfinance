-- Re-attach orphan transactions to first household account before tightening NOT NULL.
UPDATE transactions t
SET account_id = (
    SELECT a.id FROM accounts a
    WHERE a.household_id = t.household_id
    ORDER BY a.created_at
    LIMIT 1
)
WHERE t.account_id IS NULL;

ALTER TABLE transactions
    ALTER COLUMN account_id SET NOT NULL;

DROP INDEX IF EXISTS idx_accounts_member;

ALTER TABLE accounts
    DROP COLUMN IF EXISTS member_id;
