package settlement_test

import (
	"testing"

	"github.com/hunterwilliams/familyfinance/internal/domain/money"
	"github.com/hunterwilliams/familyfinance/internal/domain/settlement"
	"github.com/stretchr/testify/assert"
)

func TestCalculatePositions_MemberPaidMoreThanFairShare(t *testing.T) {
	lines := []settlement.Line{
		{
			PaidByMemberID: "a",
			PaidAmount:     money.THB(3000),
			Shares: map[string]money.Satang{
				"a": money.THB(1500),
				"b": money.THB(1500),
			},
		},
	}
	positions := settlement.CalculatePositions(lines)
	assert.Equal(t, money.THB(1500), positions["a"].Net)
	assert.Equal(t, money.THB(-1500), positions["b"].Net)
}

func TestCalculatePositions_7030Override(t *testing.T) {
	lines := []settlement.Line{
		{
			PaidByMemberID: "a",
			PaidAmount:     money.THB(1000),
			Shares: map[string]money.Satang{
				"a": money.THB(700),
				"b": money.THB(300),
			},
		},
	}
	positions := settlement.CalculatePositions(lines)
	assert.Equal(t, money.THB(300), positions["a"].Net)
	assert.Equal(t, money.THB(-300), positions["b"].Net)
}

func TestSimplifyDebts_TwoParty(t *testing.T) {
	positions := map[string]settlement.Position{
		"a": {MemberID: "a", Net: money.THB(1000)},
		"b": {MemberID: "b", Net: money.THB(-1000)},
	}
	debts := settlement.SimplifyDebts(positions)
	assert.Len(t, debts, 1)
	assert.Equal(t, "b", debts[0].FromMemberID)
	assert.Equal(t, "a", debts[0].ToMemberID)
	assert.Equal(t, money.THB(1000), debts[0].Amount)
}

func TestSimplifyDebts_ThreePartyChain(t *testing.T) {
	positions := map[string]settlement.Position{
		"a": {MemberID: "a", Net: money.THB(100)},
		"b": {MemberID: "b", Net: money.THB(0)},
		"c": {MemberID: "c", Net: money.THB(-100)},
	}
	debts := settlement.SimplifyDebts(positions)
	assert.Len(t, debts, 1)
	assert.Equal(t, "c", debts[0].FromMemberID)
	assert.Equal(t, "a", debts[0].ToMemberID)
}
