package split

import "github.com/hunterwilliams/familyfinance/internal/domain/money"

// Share is one member's portion of a transaction.
type Share struct {
	MemberID string
	Amount   money.Satang
}

// DefaultSplit divides total equally among member IDs using integer satang.
// Remainder satang goes to the first member.
func DefaultSplit(total money.Satang, memberIDs []string, parts []int) []Share {
	if len(memberIDs) == 0 {
		return nil
	}

	if len(parts) == 0 || len(parts) != len(memberIDs) {
		parts = make([]int, len(memberIDs))
		for i := range parts {
			parts[i] = 1
		}
	}

	sumParts := 0
	for _, p := range parts {
		sumParts += p
	}
	if sumParts == 0 {
		return nil
	}

	shares := make([]Share, len(memberIDs))
	allocated := money.Satang(0)
	for i, id := range memberIDs {
		if i == len(memberIDs)-1 {
			shares[i] = Share{MemberID: id, Amount: total - allocated}
			continue
		}
		amount := money.Satang(int64(total) * int64(parts[i]) / int64(sumParts))
		shares[i] = Share{MemberID: id, Amount: amount}
		allocated += amount
	}
	return shares
}
