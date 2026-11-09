package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/ai"
	"github.com/hunterwilliams/familyfinance/internal/domain/namematch"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/hunterwilliams/familyfinance/internal/service"
)

type Router struct {
	Svc *service.Service
}

type ExpenseDraft struct {
	AmountSatang   int64  `json:"amount_satang"`
	Description    string `json:"description"`
	Date           string `json:"date"`
	IsJoint        bool   `json:"is_joint"`
	AccountID      string `json:"account_id"`
	PaidByMemberID string `json:"paid_by_member_id"`
}

type HoursDraft struct {
	CaregiverName string  `json:"caregiver_name"`
	Date          string  `json:"date"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	Hours         float64 `json:"hours"`
}

func (r *Router) HandleText(ctx context.Context, lineUserID, chatID, text, openaiKey, openaiModel string) (string, error) {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)

	if isConfirm(lower) {
		return r.confirm(ctx, lineUserID)
	}
	if isCancel(lower) {
		id, _, _, err := r.Svc.Repo().LatestPending(ctx, lineUserID)
		if err != nil {
			return "", err
		}
		if id != uuid.Nil {
			_ = r.Svc.Repo().DeletePending(ctx, id)
		}
		return "Cancelled.", nil
	}

	if strings.HasPrefix(lower, "/bind") {
		h, err := r.Svc.EnsureHousehold(ctx)
		if err != nil {
			return "", err
		}
		if chatID != "" && chatID != lineUserID {
			_ = r.Svc.Repo().BindLineGroup(ctx, chatID, h.ID)
			return "This group is bound to the household.", nil
		}
		return "Use /bind in a group chat.", nil
	}

	ident, err := r.ensureIdentity(ctx, lineUserID)
	if err != nil {
		return "", err
	}
	if ident.MemberID == nil && !strings.HasPrefix(lower, "/iam") {
		members, _ := r.Svc.ListMembers(ctx)
		var b strings.Builder
		b.WriteString("I don't know who you are yet. Reply with /iam <number>:\n")
		for i, m := range members {
			fmt.Fprintf(&b, "%d. %s\n", i+1, m.Name)
		}
		return b.String(), nil
	}
	if strings.HasPrefix(lower, "/iam") {
		return r.bindMember(ctx, lineUserID, strings.TrimSpace(text[4:]))
	}

	client := ai.New(openaiKey, openaiModel)
	intent, err := client.ParseIntent(ctx, text)
	if err != nil {
		return "I could not parse that. Try: spent 200 on groceries (joint) or Nanny A extra Friday 17:00-18:00", nil
	}

	switch intent.Kind {
	case "add_expense":
		date := intent.Date
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		accounts, _ := r.Svc.ListAccounts(ctx)
		accID := ""
		if len(accounts) > 0 {
			accID = accounts[0].ID.String()
		}
		draft := ExpenseDraft{
			AmountSatang: int64(intent.AmountTHB * 100),
			Description:  intent.Description,
			Date:         date,
			IsJoint:      intent.IsJoint,
			AccountID:    accID,
		}
		_, err := r.Svc.Repo().SavePending(ctx, lineUserID, chatID, "expense", draft, time.Now().Add(15*time.Minute))
		if err != nil {
			return "", err
		}
		joint := ""
		if draft.IsJoint {
			joint = " (joint, no reimbursement)"
		}
		return fmt.Sprintf("Confirm expense: ฿%.2f — %s on %s%s?\nReply yes or no.", intent.AmountTHB, draft.Description, draft.Date, joint), nil
	case "nanny_hours":
		date := intent.Date
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		draft := HoursDraft{
			CaregiverName: intent.Caregiver,
			Date:          date,
			StartTime:     intent.StartTime,
			EndTime:       intent.EndTime,
			Hours:         intent.Hours,
		}
		_, err := r.Svc.Repo().SavePending(ctx, lineUserID, chatID, "nanny_hours", draft, time.Now().Add(15*time.Minute))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Confirm extra hours for %s on %s (%s-%s, %.2fh)? Reply yes or no.", draft.CaregiverName, draft.Date, draft.StartTime, draft.EndTime, draft.Hours), nil
	case "help":
		return "Send a receipt photo, or text like:\n- spent 350 on 7-Eleven (joint)\n- Nanny A extra Friday 17:00-18:00\nReply yes/no to confirm.", nil
	default:
		return "I didn't understand. Send a receipt photo or describe an expense. Reply help for examples.", nil
	}
}

func (r *Router) HandleImage(ctx context.Context, lineUserID, chatID, openaiKey, openaiModel, b64, mime string) (string, error) {
	if _, err := r.ensureIdentity(ctx, lineUserID); err != nil {
		return "", err
	}
	members, _ := r.Svc.ListMembers(ctx)
	client := ai.New(openaiKey, openaiModel)
	hint := ""
	for i, m := range members {
		if i > 0 {
			hint += "; "
		}
		hint += m.Name
		if m.NameTh != "" {
			hint += " / " + m.NameTh
		}
	}
	fields, err := client.ParseReceiptImage(ctx, b64, mime, hint)
	if err != nil {
		return "Could not read that image. Try again or type the amount.", nil
	}
	date := fields.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	hit := namematch.MatchAll(append([]string{fields.AccountHolder}, fields.NamesVisible...), members)
	accounts, _ := r.Svc.ListAccounts(ctx)
	accID := ""
	for _, a := range accounts {
		if hit.IsJoint || hit.MemberID == nil {
			if a.MemberID == nil {
				accID = a.ID.String()
				break
			}
		} else if a.MemberID != nil && *a.MemberID == *hit.MemberID {
			accID = a.ID.String()
			break
		}
	}
	if accID == "" && len(accounts) > 0 {
		accID = accounts[0].ID.String()
	}
	draft := ExpenseDraft{
		AmountSatang: int64(fields.AmountTHB * 100),
		Description:  fields.Merchant,
		Date:         date,
		IsJoint:      hit.IsJoint,
		AccountID:    accID,
	}
	if hit.MemberID != nil {
		draft.PaidByMemberID = hit.MemberID.String()
	}
	_, err = r.Svc.Repo().SavePending(ctx, lineUserID, chatID, "expense", draft, time.Now().Add(15*time.Minute))
	if err != nil {
		return "", err
	}
	who := "joint"
	if hit.MemberID != nil {
		for _, m := range members {
			if m.ID == *hit.MemberID {
				who = m.Name
				break
			}
		}
	} else if !hit.IsJoint {
		who = "unassigned"
	}
	return fmt.Sprintf("I read ฿%.2f at %s on %s (%s). Reply yes to save or no to cancel.", fields.AmountTHB, fields.Merchant, date, who), nil
}

func (r *Router) confirm(ctx context.Context, lineUserID string) (string, error) {
	id, kind, payload, err := r.Svc.Repo().LatestPending(ctx, lineUserID)
	if err != nil {
		return "", err
	}
	if id == uuid.Nil {
		return "Nothing to confirm.", nil
	}
	defer r.Svc.Repo().DeletePending(ctx, id)

	switch kind {
	case "expense":
		var d ExpenseDraft
		if err := json.Unmarshal(payload, &d); err != nil {
			return "Could not read pending expense.", nil
		}
		date, _ := time.Parse("2006-01-02", d.Date)
		var accountID *uuid.UUID
		if d.AccountID != "" {
			if id, err := uuid.Parse(d.AccountID); err == nil {
				accountID = &id
			}
		}
		var paidBy *uuid.UUID
		if d.PaidByMemberID != "" {
			if id, err := uuid.Parse(d.PaidByMemberID); err == nil {
				paidBy = &id
			}
		} else if !d.IsJoint {
			ident, _ := r.Svc.Repo().GetLineIdentity(ctx, lineUserID)
			if ident != nil && ident.MemberID != nil {
				paidBy = ident.MemberID
			}
		}
		_, err := r.Svc.CreateTransaction(ctx, service.CreateTransactionRequest{
			AccountID:       accountID,
			PaidByMemberID:  paidBy,
			IsJoint:         d.IsJoint,
			TransactionDate: date,
			Description:     d.Description,
			AmountSatang:    d.AmountSatang,
		})
		if err != nil {
			return "Failed to save expense: " + err.Error(), nil
		}
		return "Saved.", nil
	case "nanny_hours":
		var d HoursDraft
		if err := json.Unmarshal(payload, &d); err != nil {
			return "Could not read pending hours.", nil
		}
		h, err := r.Svc.EnsureHousehold(ctx)
		if err != nil {
			return "", err
		}
		cg, err := r.Svc.Repo().FindCaregiverByName(ctx, h.ID, d.CaregiverName)
		if err != nil {
			return "", err
		}
		if cg == nil {
			return fmt.Sprintf("No caregiver named %s. Add them in the web app first.", d.CaregiverName), nil
		}
		workDate, _ := time.Parse("2006-01-02", d.Date)
		var start, end *string
		if d.StartTime != "" {
			start = &d.StartTime
		}
		if d.EndTime != "" {
			end = &d.EndTime
		}
		_, err = r.Svc.CreateHourEntry(ctx, models.CaregiverHourEntry{
			CaregiverID: cg.ID,
			WorkDate:    workDate,
			StartTime:   start,
			EndTime:     end,
			Hours:       d.Hours,
		})
		if err != nil {
			return "Failed to save hours: " + err.Error(), nil
		}
		return "Hours saved.", nil
	default:
		return "Unknown confirmation.", nil
	}
}

func (r *Router) ensureIdentity(ctx context.Context, lineUserID string) (*models.LineIdentity, error) {
	ident, err := r.Svc.Repo().GetLineIdentity(ctx, lineUserID)
	if err != nil {
		return nil, err
	}
	if ident != nil {
		return ident, nil
	}
	h, err := r.Svc.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	id := models.LineIdentity{LineUserID: lineUserID, HouseholdID: h.ID}
	if err := r.Svc.Repo().UpsertLineIdentity(ctx, id); err != nil {
		return nil, err
	}
	return r.Svc.Repo().GetLineIdentity(ctx, lineUserID)
}

func (r *Router) bindMember(ctx context.Context, lineUserID, num string) (string, error) {
	members, err := r.Svc.ListMembers(ctx)
	if err != nil {
		return "", err
	}
	var idx int
	if _, err := fmt.Sscanf(num, "%d", &idx); err != nil || idx < 1 || idx > len(members) {
		return "Reply /iam 1 or /iam 2 with a valid number.", nil
	}
	m := members[idx-1]
	if err := r.Svc.Repo().BindLineMember(ctx, lineUserID, m.ID); err != nil {
		return "", err
	}
	return "Linked to " + m.Name + ".", nil
}

func isConfirm(s string) bool {
	s = strings.TrimSpace(s)
	return s == "yes" || s == "y" || s == "confirm"
}

func isCancel(s string) bool {
	s = strings.TrimSpace(s)
	return s == "no" || s == "n" || s == "cancel"
}

func EncodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
