ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS member_id UUID REFERENCES members(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_member ON accounts(member_id);

ALTER TABLE transactions
    ALTER COLUMN account_id DROP NOT NULL;
