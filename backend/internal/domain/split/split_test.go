package split_test

import (
	"testing"

	"github.com/hunterwilliams/familyfinance/internal/domain/money"
	"github.com/hunterwilliams/familyfinance/internal/domain/split"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSplit_5050(t *testing.T) {
	shares := split.DefaultSplit(money.THB(1000), []string{"a", "b"}, nil)
	assert.Len(t, shares, 2)
	assert.Equal(t, money.THB(500), shares[0].Amount)
	assert.Equal(t, money.THB(500), shares[1].Amount)
}

func TestDefaultSplit_7030(t *testing.T) {
	shares := split.DefaultSplit(money.THB(1000), []string{"a", "b"}, []int{7, 3})
	assert.Equal(t, money.THB(700), shares[0].Amount)
	assert.Equal(t, money.THB(300), shares[1].Amount)
}

func TestDefaultSplit_OddAmount(t *testing.T) {
	shares := split.DefaultSplit(money.THB(100.01), []string{"a", "b"}, nil)
	total := shares[0].Amount + shares[1].Amount
	assert.Equal(t, money.THB(100.01), total)
}
