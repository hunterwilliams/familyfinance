package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/domain/balance"
	"github.com/hunterwilliams/familyfinance/internal/domain/money"
	"github.com/hunterwilliams/familyfinance/internal/domain/settlement"
	"github.com/hunterwilliams/familyfinance/internal/domain/split"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/hunterwilliams/familyfinance/internal/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureHousehold(ctx context.Context) (models.Household, error) {
	h, err := s.repo.GetOrCreateDefaultHousehold(ctx)
	if err != nil {
		return h, err
	}
	if err := s.repo.SeedDefaults(ctx, h.ID); err != nil {
		return h, err
	}
	return h, nil
}

func (s *Service) ListMembers(ctx context.Context) ([]models.Member, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, h.ID)
}

func (s *Service) CreateMember(ctx context.Context, name string) (models.Member, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Member{}, err
	}
	return s.repo.CreateMember(ctx, h.ID, name)
}

func (s *Service) UpdateMember(ctx context.Context, id uuid.UUID, name, nameTh string) (models.Member, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Member{}, err
	}
	return s.repo.UpdateMember(ctx, h.ID, id, name, nameTh)
}

func (s *Service) ListAccounts(ctx context.Context) ([]models.Account, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAccounts(ctx, h.ID)
}

func (s *Service) CreateAccount(ctx context.Context, a models.Account) (models.Account, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Account{}, err
	}
	a.HouseholdID = h.ID
	if a.MemberID != nil {
		a.Type = models.AccountPersonal
	} else if a.Type == "" {
		a.Type = models.AccountJointChecking
	}
	return s.repo.CreateAccount(ctx, a)
}

func (s *Service) ListCategories(ctx context.Context) ([]models.Category, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCategories(ctx, h.ID)
}

func (s *Service) CreateCategory(ctx context.Context, c models.Category) (models.Category, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Category{}, err
	}
	c.HouseholdID = h.ID
	return s.repo.CreateCategory(ctx, c)
}

func (s *Service) ListTransactions(ctx context.Context) ([]models.Transaction, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListTransactions(ctx, h.ID)
}

type CreateTransactionRequest struct {
	AccountID       *uuid.UUID
	CategoryID      *uuid.UUID
	PaidByMemberID  *uuid.UUID
	IsJoint         bool
	TransactionDate time.Time
	Description     string
	AmountSatang    int64
	ReceiptPath     *string
	SplitParts      []int
	SplitMemberIDs  []uuid.UUID
}

func (s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (models.Transaction, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Transaction{}, err
	}

	var splits []models.Split
	if !req.IsJoint {
		members, err := s.repo.ListMembers(ctx, h.ID)
		if err != nil {
			return models.Transaction{}, err
		}

		memberIDs := req.SplitMemberIDs
		if len(memberIDs) == 0 {
			for _, m := range members {
				memberIDs = append(memberIDs, m.ID)
			}
		}

		parts := req.SplitParts
		if len(parts) == 0 {
			parts = defaultSplitParts(len(memberIDs), h.DefaultSplitParts)
		}

		splitShares := split.DefaultSplit(money.Satang(req.AmountSatang), uuidStrings(memberIDs), parts)
		for i, sh := range splitShares {
			splits = append(splits, models.Split{
				MemberID:     memberIDs[i],
				AmountSatang: int64(sh.Amount),
			})
		}
	}

	return s.repo.CreateTransaction(ctx, repository.CreateTransactionInput{
		HouseholdID:     h.ID,
		AccountID:       req.AccountID,
		CategoryID:      req.CategoryID,
		PaidByMemberID:  req.PaidByMemberID,
		IsJoint:         req.IsJoint,
		TransactionDate: req.TransactionDate,
		Description:     req.Description,
		AmountSatang:    req.AmountSatang,
		ReceiptPath:     req.ReceiptPath,
		Splits:          splits,
	})
}

type BulkUpdateItem struct {
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

func (s *Service) BulkUpdateTransactions(ctx context.Context, items []BulkUpdateItem) ([]models.Transaction, error) {
	out := make([]models.Transaction, 0)
	for _, item := range items {
		t, err := s.repo.UpdateTransaction(ctx, repository.UpdateTransactionInput{
			ID:              item.ID,
			AccountID:       item.AccountID,
			CategoryID:      item.CategoryID,
			PaidByMemberID:  item.PaidByMemberID,
			IsJoint:         item.IsJoint,
			TransactionDate: item.TransactionDate,
			Description:     item.Description,
			AmountSatang:    item.AmountSatang,
			Splits:          item.Splits,
			UpdateSplits:    item.UpdateSplits,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

type SettlementReport struct {
	Month     string                        `json:"month"`
	Positions map[string]settlement.Position `json:"positions"`
	Debts     []settlement.Debt              `json:"debts"`
}

func (s *Service) SettlementReport(ctx context.Context, month string) (SettlementReport, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return SettlementReport{}, err
	}

	start, err := repository.ParseMonth(month)
	if err != nil {
		return SettlementReport{}, err
	}
	end := start.AddDate(0, 1, 0)

	txns, err := s.repo.ListTransactions(ctx, h.ID)
	if err != nil {
		return SettlementReport{}, err
	}

	var lines []settlement.Line
	for _, t := range txns {
		if t.IsJoint {
			continue
		}
		if t.TransactionDate.Before(start) || !t.TransactionDate.Before(end) {
			continue
		}
		shares := map[string]money.Satang{}
		for _, sp := range t.Splits {
			shares[sp.MemberID.String()] = money.Satang(sp.AmountSatang)
		}
		paidBy := ""
		if t.PaidByMemberID != nil {
			paidBy = t.PaidByMemberID.String()
		}
		lines = append(lines, settlement.Line{
			PaidByMemberID: paidBy,
			PaidAmount:     money.Satang(t.AmountSatang),
			Shares:         shares,
		})
	}

	positions := settlement.CalculatePositions(lines)
	debts := settlement.SimplifyDebts(positions)
	if debts == nil {
		debts = []settlement.Debt{}
	}

	return SettlementReport{
		Month:     month,
		Positions: positions,
		Debts:     debts,
	}, nil
}

type EffectiveBalanceReport struct {
	EffectiveBalanceSatang int64 `json:"effective_balance_satang"`
	SnapshotBalanceSatang  int64 `json:"snapshot_balance_satang"`
	PendingCCSatang        int64 `json:"pending_cc_satang"`
	SettlementsSinceSatang int64 `json:"settlements_since_satang"`
	GapSatang              int64 `json:"gap_satang"`
}

func (s *Service) EffectiveBalance(ctx context.Context, checkingAccountID uuid.UUID, ccAccountID *uuid.UUID) (EffectiveBalanceReport, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return EffectiveBalanceReport{}, err
	}

	snap, err := s.repo.LatestBalanceSnapshot(ctx, checkingAccountID)
	if err != nil {
		return EffectiveBalanceReport{}, err
	}

	var snapshotSatang int64
	var since time.Time
	if snap != nil {
		snapshotSatang = snap.BalanceSatang
		since = snap.SnapshotDate
	}

	var pendingCC int64
	if ccAccountID != nil && !since.IsZero() {
		pendingCC, err = s.repo.SumCCChargesSince(ctx, *ccAccountID, since)
		if err != nil {
			return EffectiveBalanceReport{}, err
		}
	}

	settlementsSince := int64(0)
	if !since.IsZero() {
		settlementsSince, err = s.repo.SumSettlementsSince(ctx, h.ID, since)
		if err != nil {
			return EffectiveBalanceReport{}, err
		}
	}

	effective := balance.EffectiveCheckingBalance(balance.Input{
		LatestSnapshot:   money.Satang(snapshotSatang),
		PendingCCCharges: money.Satang(pendingCC),
		SettlementsSince: money.Satang(settlementsSince),
	})

	txns, err := s.repo.ListTransactions(ctx, h.ID)
	if err != nil {
		return EffectiveBalanceReport{}, err
	}
	var ledger int64
	for _, t := range txns {
		if t.AccountID != nil && *t.AccountID == checkingAccountID {
			ledger += t.AmountSatang
		}
	}
	recon := balance.Reconcile(money.Satang(snapshotSatang), money.Satang(ledger))

	return EffectiveBalanceReport{
		EffectiveBalanceSatang: int64(effective),
		SnapshotBalanceSatang:  snapshotSatang,
		PendingCCSatang:        pendingCC,
		SettlementsSinceSatang: settlementsSince,
		GapSatang:              int64(recon.Gap),
	}, nil
}

func (s *Service) ListBalanceSnapshots(ctx context.Context, accountID uuid.UUID) ([]models.BalanceSnapshot, error) {
	return s.repo.ListBalanceSnapshots(ctx, accountID)
}

func (s *Service) CreateBalanceSnapshot(ctx context.Context, b models.BalanceSnapshot) (models.BalanceSnapshot, error) {
	return s.repo.CreateBalanceSnapshot(ctx, b)
}

func (s *Service) ListCCBalances(ctx context.Context, accountID uuid.UUID) ([]models.CCStatementBalance, error) {
	return s.repo.ListCCBalances(ctx, accountID)
}

func (s *Service) CreateCCBalance(ctx context.Context, b models.CCStatementBalance) (models.CCStatementBalance, error) {
	return s.repo.CreateCCBalance(ctx, b)
}

func (s *Service) ListSettlements(ctx context.Context) ([]models.Settlement, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSettlements(ctx, h.ID)
}

func (s *Service) CreateSettlement(ctx context.Context, st models.Settlement) (models.Settlement, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Settlement{}, err
	}
	st.HouseholdID = h.ID
	return s.repo.CreateSettlement(ctx, st)
}

func (s *Service) ListSavingsGoals(ctx context.Context) ([]models.SavingsGoal, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSavingsGoals(ctx, h.ID)
}

func (s *Service) CreateSavingsGoal(ctx context.Context, g models.SavingsGoal) (models.SavingsGoal, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	g.HouseholdID = h.ID
	return s.repo.CreateSavingsGoal(ctx, g)
}

func (s *Service) UpdateSavingsGoal(ctx context.Context, g models.SavingsGoal) (models.SavingsGoal, error) {
	return s.repo.UpdateSavingsGoal(ctx, g)
}

func defaultSplitParts(n int, householdParts []int) []int {
	if len(householdParts) == n {
		return householdParts
	}
	parts := make([]int, n)
	for i := range parts {
		parts[i] = 1
	}
	return parts
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
