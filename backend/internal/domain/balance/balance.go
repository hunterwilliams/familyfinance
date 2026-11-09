package balance

import "github.com/hunterwilliams/familyfinance/internal/domain/money"

// Input for effective checking balance calculation.
type Input struct {
	LatestSnapshot        money.Satang
	PendingCCCharges      money.Satang
	UnreimbursedPersonal  money.Satang
	SettlementsSince      money.Satang
}

// EffectiveCheckingBalance estimates actual usable joint checking balance.
func EffectiveCheckingBalance(in Input) money.Satang {
	return in.LatestSnapshot - in.PendingCCCharges - in.UnreimbursedPersonal + in.SettlementsSince
}

// Reconciliation compares snapshot to ledger-derived balance.
type Reconciliation struct {
	SnapshotBalance money.Satang
	LedgerBalance   money.Satang
	Gap             money.Satang
}

func Reconcile(snapshot, ledger money.Satang) Reconciliation {
	return Reconciliation{
		SnapshotBalance: snapshot,
		LedgerBalance:   ledger,
		Gap:             snapshot - ledger,
	}
}
