package settlement

import "github.com/hunterwilliams/familyfinance/internal/domain/money"

// Line represents one attributed transaction for settlement.
type Line struct {
	PaidByMemberID string
	PaidAmount     money.Satang
	Shares         map[string]money.Satang // memberID -> fair share
}

// Position is a member's net position for a period.
type Position struct {
	MemberID  string       `json:"member_id"`
	Paid      money.Satang `json:"paid"`
	FairShare money.Satang `json:"fair_share"`
	Net       money.Satang `json:"net"` // positive = creditor (owed money), negative = debtor
}

// Debt is a simplified owed amount from one member to another.
type Debt struct {
	FromMemberID string       `json:"from_member_id"`
	ToMemberID   string       `json:"to_member_id"`
	Amount       money.Satang `json:"amount"`
}

// CalculatePositions aggregates paid vs fair share per member.
func CalculatePositions(lines []Line) map[string]Position {
	positions := map[string]Position{}

	add := func(id string, paid, fair money.Satang) {
		p := positions[id]
		p.MemberID = id
		p.Paid += paid
		p.FairShare += fair
		p.Net = p.Paid - p.FairShare
		positions[id] = p
	}

	for _, line := range lines {
		add(line.PaidByMemberID, line.PaidAmount, 0)
		for memberID, share := range line.Shares {
			add(memberID, 0, share)
		}
	}
	return positions
}

// SimplifyDebts converts net positions into minimum transfer debts.
func SimplifyDebts(positions map[string]Position) []Debt {
	type bucket struct {
		id   string
		net  money.Satang
	}

	var creditors, debtors []bucket
	for _, p := range positions {
		if p.Net > 0 {
			creditors = append(creditors, bucket{id: p.MemberID, net: p.Net})
		} else if p.Net < 0 {
			debtors = append(debtors, bucket{id: p.MemberID, net: -p.Net})
		}
	}

	var debts []Debt
	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		amount := creditors[i].net
		if debtors[j].net < amount {
			amount = debtors[j].net
		}
		if amount > 0 {
			debts = append(debts, Debt{
				FromMemberID: debtors[j].id,
				ToMemberID:   creditors[i].id,
				Amount:       amount,
			})
		}
		creditors[i].net -= amount
		debtors[j].net -= amount
		if creditors[i].net == 0 {
			i++
		}
		if debtors[j].net == 0 {
			j++
		}
	}
	return debts
}
