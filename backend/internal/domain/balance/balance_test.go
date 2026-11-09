package balance_test

import (
	"testing"

	"github.com/hunterwilliams/familyfinance/internal/domain/balance"
	"github.com/hunterwilliams/familyfinance/internal/domain/money"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveCheckingBalance(t *testing.T) {
	result := balance.EffectiveCheckingBalance(balance.Input{
		LatestSnapshot:       money.THB(50000),
		PendingCCCharges:     money.THB(5000),
		UnreimbursedPersonal: money.THB(2000),
		SettlementsSince:     money.THB(1000),
	})
	assert.Equal(t, money.THB(44000), result)
}

func TestReconcile(t *testing.T) {
	r := balance.Reconcile(money.THB(10000), money.THB(9800))
	assert.Equal(t, money.THB(200), r.Gap)
}
