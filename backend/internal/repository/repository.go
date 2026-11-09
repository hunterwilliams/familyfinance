package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) GetOrCreateDefaultHousehold(ctx context.Context) (models.Household, error) {
	var h models.Household
	var partsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, default_split_parts, created_at
		FROM households ORDER BY created_at LIMIT 1
	`).Scan(&h.ID, &h.Name, &partsJSON, &h.CreatedAt)
	if err == nil {
		_ = json.Unmarshal(partsJSON, &h.DefaultSplitParts)
		return h, nil
	}
	if err != pgx.ErrNoRows {
		return h, err
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO households (name, default_split_parts)
		VALUES ('Family', '[1,1]'::jsonb)
		RETURNING id, name, default_split_parts, created_at
	`).Scan(&h.ID, &h.Name, &partsJSON, &h.CreatedAt)
	if err == nil {
		_ = json.Unmarshal(partsJSON, &h.DefaultSplitParts)
	}
	return h, err
}

func (r *Repository) ListMembers(ctx context.Context, householdID uuid.UUID) ([]models.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, COALESCE(name_th, ''), created_at FROM members
		WHERE household_id = $1 ORDER BY name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.Member, 0)
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.ID, &m.HouseholdID, &m.Name, &m.NameTh, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) CreateMember(ctx context.Context, householdID uuid.UUID, name string) (models.Member, error) {
	var m models.Member
	err := r.pool.QueryRow(ctx, `
		INSERT INTO members (household_id, name) VALUES ($1, $2)
		RETURNING id, household_id, name, COALESCE(name_th, ''), created_at
	`, householdID, name).Scan(&m.ID, &m.HouseholdID, &m.Name, &m.NameTh, &m.CreatedAt)
	return m, err
}

func (r *Repository) UpdateMember(ctx context.Context, householdID, id uuid.UUID, name, nameTh string) (models.Member, error) {
	var m models.Member
	err := r.pool.QueryRow(ctx, `
		UPDATE members SET name = $3, name_th = $4
		WHERE id = $1 AND household_id = $2
		RETURNING id, household_id, name, COALESCE(name_th, ''), created_at
	`, id, householdID, name, nameTh).Scan(&m.ID, &m.HouseholdID, &m.Name, &m.NameTh, &m.CreatedAt)
	return m, err
}

func (r *Repository) ListAccounts(ctx context.Context, householdID uuid.UUID) ([]models.Account, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, member_id, name, type, COALESCE(institution, ''), created_at
		FROM accounts WHERE household_id = $1 ORDER BY name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.Account, 0)
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.HouseholdID, &a.MemberID, &a.Name, &a.Type, &a.Institution, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) CreateAccount(ctx context.Context, a models.Account) (models.Account, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO accounts (household_id, member_id, name, type, institution)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		RETURNING id, household_id, member_id, name, type, COALESCE(institution, ''), created_at
	`, a.HouseholdID, a.MemberID, a.Name, a.Type, a.Institution).Scan(
		&a.ID, &a.HouseholdID, &a.MemberID, &a.Name, &a.Type, &a.Institution, &a.CreatedAt,
	)
	return a, err
}

func (r *Repository) ListCategories(ctx context.Context, householdID uuid.UUID) ([]models.Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, parent_id, created_at
		FROM categories WHERE household_id = $1 ORDER BY name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.Category, 0)
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.HouseholdID, &c.Name, &c.ParentID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO categories (household_id, name, parent_id)
		VALUES ($1, $2, $3)
		RETURNING id, household_id, name, parent_id, created_at
	`, c.HouseholdID, c.Name, c.ParentID).Scan(
		&c.ID, &c.HouseholdID, &c.Name, &c.ParentID, &c.CreatedAt,
	)
	return c, err
}

func (r *Repository) ListTransactions(ctx context.Context, householdID uuid.UUID) ([]models.Transaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, account_id, category_id, paid_by_member_id, is_joint,
		       transaction_date, description, amount_satang, receipt_path, created_at, updated_at
		FROM transactions WHERE household_id = $1 ORDER BY transaction_date DESC, created_at DESC
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.Transaction, 0)
	var ids []uuid.UUID
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.ID, &t.HouseholdID, &t.AccountID, &t.CategoryID, &t.PaidByMemberID, &t.IsJoint,
			&t.TransactionDate, &t.Description, &t.AmountSatang, &t.ReceiptPath,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return out, nil
	}
	splitMap, err := r.getSplitsForTransactions(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Splits = splitMap[out[i].ID]
	}
	return out, nil
}

func (r *Repository) getSplitsForTransactions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]models.Split, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, transaction_id, member_id, amount_satang
		FROM transaction_splits WHERE transaction_id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[uuid.UUID][]models.Split)
	for rows.Next() {
		var s models.Split
		if err := rows.Scan(&s.ID, &s.TransactionID, &s.MemberID, &s.AmountSatang); err != nil {
			return nil, err
		}
		result[s.TransactionID] = append(result[s.TransactionID], s)
	}
	return result, rows.Err()
}

type CreateTransactionInput struct {
	HouseholdID     uuid.UUID
	AccountID       *uuid.UUID
	CategoryID      *uuid.UUID
	PaidByMemberID  *uuid.UUID
	IsJoint         bool
	TransactionDate time.Time
	Description     string
	AmountSatang    int64
	ReceiptPath     *string
	Splits          []models.Split
}

func (r *Repository) CreateTransaction(ctx context.Context, in CreateTransactionInput) (models.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Transaction{}, err
	}
	defer tx.Rollback(ctx)

	var t models.Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (
			household_id, account_id, category_id, paid_by_member_id, is_joint,
			transaction_date, description, amount_satang, receipt_path
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, household_id, account_id, category_id, paid_by_member_id, is_joint,
		          transaction_date, description, amount_satang, receipt_path, created_at, updated_at
	`, in.HouseholdID, in.AccountID, in.CategoryID, in.PaidByMemberID, in.IsJoint,
		in.TransactionDate, in.Description, in.AmountSatang, in.ReceiptPath,
	).Scan(
		&t.ID, &t.HouseholdID, &t.AccountID, &t.CategoryID, &t.PaidByMemberID, &t.IsJoint,
		&t.TransactionDate, &t.Description, &t.AmountSatang, &t.ReceiptPath,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}

	for _, s := range in.Splits {
		var split models.Split
		err = tx.QueryRow(ctx, `
			INSERT INTO transaction_splits (transaction_id, member_id, amount_satang)
			VALUES ($1, $2, $3)
			RETURNING id, transaction_id, member_id, amount_satang
		`, t.ID, s.MemberID, s.AmountSatang).Scan(
			&split.ID, &split.TransactionID, &split.MemberID, &split.AmountSatang,
		)
		if err != nil {
			return t, err
		}
		t.Splits = append(t.Splits, split)
	}

	if err := tx.Commit(ctx); err != nil {
		return t, err
	}
	return t, nil
}

type UpdateTransactionInput struct {
	ID              uuid.UUID
	AccountID       *uuid.UUID
	CategoryID      *uuid.UUID
	PaidByMemberID  *uuid.UUID
	IsJoint         *bool
	TransactionDate *time.Time
	Description     *string
	AmountSatang    *int64
	Splits          []models.Split
	UpdateSplits    bool
}

func (r *Repository) UpdateTransaction(ctx context.Context, in UpdateTransactionInput) (models.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Transaction{}, err
	}
	defer tx.Rollback(ctx)

	var t models.Transaction
	err = tx.QueryRow(ctx, `
		UPDATE transactions SET
			account_id = COALESCE($2, account_id),
			category_id = COALESCE($3, category_id),
			paid_by_member_id = CASE WHEN $8 = true THEN NULL ELSE COALESCE($4, paid_by_member_id) END,
			transaction_date = COALESCE($5, transaction_date),
			description = COALESCE($6, description),
			amount_satang = COALESCE($7, amount_satang),
			is_joint = COALESCE($8, is_joint),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, household_id, account_id, category_id, paid_by_member_id, is_joint,
		          transaction_date, description, amount_satang, receipt_path, created_at, updated_at
	`, in.ID, in.AccountID, in.CategoryID, in.PaidByMemberID, in.TransactionDate, in.Description, in.AmountSatang, in.IsJoint,
	).Scan(
		&t.ID, &t.HouseholdID, &t.AccountID, &t.CategoryID, &t.PaidByMemberID, &t.IsJoint,
		&t.TransactionDate, &t.Description, &t.AmountSatang, &t.ReceiptPath,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}

	if in.UpdateSplits {
		_, err = tx.Exec(ctx, `DELETE FROM transaction_splits WHERE transaction_id = $1`, t.ID)
		if err != nil {
			return t, err
		}
		for _, s := range in.Splits {
			var split models.Split
			err = tx.QueryRow(ctx, `
				INSERT INTO transaction_splits (transaction_id, member_id, amount_satang)
				VALUES ($1, $2, $3)
				RETURNING id, transaction_id, member_id, amount_satang
			`, t.ID, s.MemberID, s.AmountSatang).Scan(
				&split.ID, &split.TransactionID, &split.MemberID, &split.AmountSatang,
			)
			if err != nil {
				return t, err
			}
			t.Splits = append(t.Splits, split)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return t, err
	}
	return t, nil
}

func (r *Repository) ListBalanceSnapshots(ctx context.Context, accountID uuid.UUID) ([]models.BalanceSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, account_id, snapshot_date, balance_satang, note, created_at
		FROM balance_snapshots WHERE account_id = $1 ORDER BY snapshot_date DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBalanceSnapshots(rows)
}

func scanBalanceSnapshots(rows pgx.Rows) ([]models.BalanceSnapshot, error) {
	defer rows.Close()
	var out = make([]models.BalanceSnapshot, 0)
	for rows.Next() {
		var b models.BalanceSnapshot
		if err := rows.Scan(&b.ID, &b.AccountID, &b.SnapshotDate, &b.BalanceSatang, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) CreateBalanceSnapshot(ctx context.Context, b models.BalanceSnapshot) (models.BalanceSnapshot, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO balance_snapshots (account_id, snapshot_date, balance_satang, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, account_id, snapshot_date, balance_satang, note, created_at
	`, b.AccountID, b.SnapshotDate, b.BalanceSatang, b.Note).Scan(
		&b.ID, &b.AccountID, &b.SnapshotDate, &b.BalanceSatang, &b.Note, &b.CreatedAt,
	)
	return b, err
}

func (r *Repository) LatestBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (*models.BalanceSnapshot, error) {
	var b models.BalanceSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT id, account_id, snapshot_date, balance_satang, note, created_at
		FROM balance_snapshots WHERE account_id = $1
		ORDER BY snapshot_date DESC, created_at DESC LIMIT 1
	`, accountID).Scan(&b.ID, &b.AccountID, &b.SnapshotDate, &b.BalanceSatang, &b.Note, &b.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListCCBalances(ctx context.Context, accountID uuid.UUID) ([]models.CCStatementBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, account_id, statement_date, balance_satang, note, created_at
		FROM cc_statement_balances WHERE account_id = $1 ORDER BY statement_date DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.CCStatementBalance, 0)
	for rows.Next() {
		var b models.CCStatementBalance
		if err := rows.Scan(&b.ID, &b.AccountID, &b.StatementDate, &b.BalanceSatang, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCCBalance(ctx context.Context, b models.CCStatementBalance) (models.CCStatementBalance, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO cc_statement_balances (account_id, statement_date, balance_satang, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, account_id, statement_date, balance_satang, note, created_at
	`, b.AccountID, b.StatementDate, b.BalanceSatang, b.Note).Scan(
		&b.ID, &b.AccountID, &b.StatementDate, &b.BalanceSatang, &b.Note, &b.CreatedAt,
	)
	return b, err
}

func (r *Repository) ListSettlements(ctx context.Context, householdID uuid.UUID) ([]models.Settlement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, from_member_id, to_member_id, settlement_date, amount_satang, note, created_at
		FROM settlements WHERE household_id = $1 ORDER BY settlement_date DESC
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.Settlement, 0)
	for rows.Next() {
		var s models.Settlement
		if err := rows.Scan(&s.ID, &s.HouseholdID, &s.FromMemberID, &s.ToMemberID,
			&s.SettlementDate, &s.AmountSatang, &s.Note, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) CreateSettlement(ctx context.Context, s models.Settlement) (models.Settlement, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO settlements (household_id, from_member_id, to_member_id, settlement_date, amount_satang, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, household_id, from_member_id, to_member_id, settlement_date, amount_satang, note, created_at
	`, s.HouseholdID, s.FromMemberID, s.ToMemberID, s.SettlementDate, s.AmountSatang, s.Note).Scan(
		&s.ID, &s.HouseholdID, &s.FromMemberID, &s.ToMemberID,
		&s.SettlementDate, &s.AmountSatang, &s.Note, &s.CreatedAt,
	)
	return s, err
}

func (r *Repository) ListSavingsGoals(ctx context.Context, householdID uuid.UUID) ([]models.SavingsGoal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, target_satang, current_satang, deadline, category_id, created_at
		FROM savings_goals WHERE household_id = $1 ORDER BY created_at
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out = make([]models.SavingsGoal, 0)
	for rows.Next() {
		var g models.SavingsGoal
		if err := rows.Scan(&g.ID, &g.HouseholdID, &g.Name, &g.TargetSatang, &g.CurrentSatang,
			&g.Deadline, &g.CategoryID, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) CreateSavingsGoal(ctx context.Context, g models.SavingsGoal) (models.SavingsGoal, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO savings_goals (household_id, name, target_satang, current_satang, deadline, category_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, household_id, name, target_satang, current_satang, deadline, category_id, created_at
	`, g.HouseholdID, g.Name, g.TargetSatang, g.CurrentSatang, g.Deadline, g.CategoryID).Scan(
		&g.ID, &g.HouseholdID, &g.Name, &g.TargetSatang, &g.CurrentSatang,
		&g.Deadline, &g.CategoryID, &g.CreatedAt,
	)
	return g, err
}

func (r *Repository) UpdateSavingsGoal(ctx context.Context, g models.SavingsGoal) (models.SavingsGoal, error) {
	err := r.pool.QueryRow(ctx, `
		UPDATE savings_goals SET
			name = $2, target_satang = $3, current_satang = $4, deadline = $5, category_id = $6
		WHERE id = $1
		RETURNING id, household_id, name, target_satang, current_satang, deadline, category_id, created_at
	`, g.ID, g.Name, g.TargetSatang, g.CurrentSatang, g.Deadline, g.CategoryID).Scan(
		&g.ID, &g.HouseholdID, &g.Name, &g.TargetSatang, &g.CurrentSatang,
		&g.Deadline, &g.CategoryID, &g.CreatedAt,
	)
	return g, err
}

func (r *Repository) SumCCChargesSince(ctx context.Context, accountID uuid.UUID, since time.Time) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_satang), 0) FROM transactions
		WHERE account_id = $1 AND transaction_date > $2
	`, accountID, since).Scan(&total)
	return total, err
}

func (r *Repository) SumSettlementsSince(ctx context.Context, householdID uuid.UUID, since time.Time) (int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_satang), 0) FROM settlements
		WHERE household_id = $1 AND settlement_date > $2
	`, householdID, since).Scan(&total)
	return total, err
}

func (r *Repository) SeedDefaults(ctx context.Context, householdID uuid.UUID) error {
	count := 0
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM members WHERE household_id = $1`, householdID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	members := []string{"Member 1", "Member 2"}
	var memberIDs []uuid.UUID
	for _, name := range members {
		m, err := r.CreateMember(ctx, householdID, name)
		if err != nil {
			return err
		}
		memberIDs = append(memberIDs, m.ID)
	}
	_ = memberIDs

	accounts := []models.Account{
		{HouseholdID: householdID, Name: "SCB Joint Checking", Type: models.AccountJointChecking, Institution: "SCB"},
		{HouseholdID: householdID, Name: "KTC Joint Credit", Type: models.AccountJointCredit, Institution: "KTC"},
	}
	for _, a := range accounts {
		if _, err := r.CreateAccount(ctx, a); err != nil {
			return err
		}
	}

	categories := []string{"Food", "Transport", "Housing", "Utilities", "Entertainment", "Savings", "Other"}
	for _, name := range categories {
		if _, err := r.CreateCategory(ctx, models.Category{HouseholdID: householdID, Name: name}); err != nil {
			return err
		}
	}
	return nil
}

func ParseMonth(s string) (time.Time, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month format, use YYYY-MM: %w", err)
	}
	return t, nil
}
