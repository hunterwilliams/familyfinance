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
	"github.com/hunterwilliams/familyfinance/internal/domain/nanny"
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
	Kind          string  `json:"kind,omitempty"`
	Note          string  `json:"note,omitempty"`
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

	now := bangkokNow()
	cgs, _ := r.Svc.ListCaregivers(ctx)
	refs := caregiverRefs(cgs)
	paid := r.paidCalendar(ctx)
	if hours, ok := nanny.ParseHoursMessage(text, refs, paid, now); ok {
		return r.queueNannyHours(ctx, lineUserID, chatID, hours)
	}

	client := ai.New(openaiKey, openaiModel)
	intent, err := client.ParseIntent(ctx, text, nannyContext(cgs))
	if err != nil {
		return "I could not parse that. Try: spent 200 on groceries (joint) or nanny come early tomorrow at 7am", nil
	}

	switch intent.Kind {
	case "add_expense":
		date := intent.Date
		if date == "" {
			date = now.Format("2006-01-02")
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
			date = now.Format("2006-01-02")
		}
		cg := pickCaregiver(intent.Caregiver, cgs)
		hours := nanny.HoursIntent{
			CaregiverName: intent.Caregiver,
			Date:          date,
			StartTime:     intent.StartTime,
			EndTime:       intent.EndTime,
			Hours:         intent.Hours,
			Note:          "extra hours",
		}
		if cg != nil {
			hours.CaregiverName = cg.Name
			ref := caregiverRef(*cg)
			if start, end, h, ok := nanny.ApplyShiftDefaults(hours.StartTime, hours.EndTime, hours.Hours, ref); ok {
				hours.StartTime, hours.EndTime, hours.Hours = start, end, h
			}
		}
		return r.queueNannyHours(ctx, lineUserID, chatID, hours)
	case "nanny_holiday":
		date := intent.Date
		if date == "" {
			date = now.Format("2006-01-02")
		}
		cg := pickCaregiver(intent.Caregiver, cgs)
		if cg == nil {
			return "Add the nanny in the web app first, then try again.", nil
		}
		d, err := time.ParseInLocation("2006-01-02", date, now.Location())
		if err != nil {
			d = now
		}
		return r.queueNannyHours(ctx, lineUserID, chatID, nanny.HolidayFor(caregiverRef(*cg), d, paid))
	case "nanny_worked_holiday":
		date := intent.Date
		if date == "" {
			date = now.Format("2006-01-02")
		}
		cg := pickCaregiver(intent.Caregiver, cgs)
		if cg == nil {
			return "Add the nanny in the web app first, then try again.", nil
		}
		d, err := time.ParseInLocation("2006-01-02", date, now.Location())
		if err != nil {
			d = now
		}
		return r.queueNannyHours(ctx, lineUserID, chatID, nanny.WorkedHolidayFor(caregiverRef(*cg), d, paid))
	case "help":
		return "Send a receipt photo, or text like:\n- spent 350 on 7-Eleven (joint)\n- nanny come early tomorrow at 7am\n- nanny holiday tomorrow\n- nanny working tomorrow\nReply yes/no to confirm.", nil
	default:
		return "I didn't understand. Send a receipt photo or describe an expense. Reply help for examples.", nil
	}
}

func (r *Router) queueNannyHours(ctx context.Context, lineUserID, chatID string, hours nanny.HoursIntent) (string, error) {
	name := hours.CaregiverName
	if name == "" {
		name = "nanny"
	}
	if hours.Kind == nanny.KindPaidHoliday {
		label := hours.Note
		if label == "" {
			label = "paid holiday"
		}
		return fmt.Sprintf("%s is a paid holiday (%s). No hours owed unless they work that day.", hours.Date, label), nil
	}
	if hours.Kind == nanny.KindHoliday {
		if hours.Hours == 0 {
			day := hours.Date
			if d, err := time.Parse("2006-01-02", hours.Date); err == nil {
				day = d.Weekday().String()
			}
			return fmt.Sprintf("%s does not usually work on %s. Set usual days in the Nanny tab if that is wrong.", name, day), nil
		}
	} else if hours.Kind == nanny.KindWorkedHoliday && hours.Hours == 0 {
		return fmt.Sprintf("%s is not a paid holiday. Add it on the Nanny tab, or log extra hours if they stayed late.", hours.Date), nil
	} else if hours.Hours <= 0 || hours.StartTime == "" || hours.EndTime == "" {
		return "I need usual nanny hours to calculate extra time. Set the default start and end in the Nanny tab, then try again.", nil
	}
	if hours.Kind == "" {
		hours.Kind = nanny.KindExtra
	}
	draft := HoursDraft{
		CaregiverName: hours.CaregiverName,
		Date:          hours.Date,
		StartTime:     hours.StartTime,
		EndTime:       hours.EndTime,
		Hours:         hours.Hours,
		Kind:          hours.Kind,
		Note:          hours.Note,
	}
	_, err := r.Svc.Repo().SavePending(ctx, lineUserID, chatID, "nanny_hours", draft, time.Now().Add(15*time.Minute))
	if err != nil {
		return "", err
	}
	if draft.Kind == nanny.KindHoliday {
		return fmt.Sprintf("Confirm holiday for %s on %s (%s-%s, %.2fh owed, offsets extra)? Reply yes or no.", name, draft.Date, draft.StartTime, draft.EndTime, draft.Hours), nil
	}
	if draft.Kind == nanny.KindWorkedHoliday {
		return fmt.Sprintf("Confirm %s worked a paid holiday on %s (%s-%s, %.2fh extra pay)? Reply yes or no.", name, draft.Date, draft.StartTime, draft.EndTime, draft.Hours), nil
	}
	return fmt.Sprintf("Confirm extra hours for %s on %s (%s-%s, %.2fh)? Reply yes or no.", name, draft.Date, draft.StartTime, draft.EndTime, draft.Hours), nil
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
		kind := d.Kind
		if kind == "" {
			kind = nanny.KindExtra
		}
		_, err = r.Svc.CreateHourEntry(ctx, models.CaregiverHourEntry{
			CaregiverID: cg.ID,
			WorkDate:    workDate,
			StartTime:   start,
			EndTime:     end,
			Hours:       d.Hours,
			Kind:        kind,
			Note:        d.Note,
		})
		if err != nil {
			return "Failed to save hours: " + err.Error(), nil
		}
		if kind == nanny.KindHoliday {
			return "Holiday saved. It offsets extra hours.", nil
		}
		if kind == nanny.KindWorkedHoliday {
			return "Worked holiday saved as extra pay.", nil
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

func caregiverRef(c models.Caregiver) nanny.Ref {
	return nanny.Ref{
		Name:         c.Name,
		DefaultStart: derefTime(c.DefaultStartTime),
		DefaultEnd:   derefTime(c.DefaultEndTime),
		DefaultDays:  c.DefaultWeekdays,
	}
}

func caregiverRefs(cgs []models.Caregiver) []nanny.Ref {
	out := make([]nanny.Ref, 0, len(cgs))
	for _, c := range cgs {
		out = append(out, caregiverRef(c))
	}
	return out
}

func nannyContext(cgs []models.Caregiver) string {
	if len(cgs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Known nannies, usual hours, and usual days:\n")
	for _, c := range cgs {
		start := derefTime(c.DefaultStartTime)
		end := derefTime(c.DefaultEndTime)
		if start == "" {
			start = "?"
		}
		if end == "" {
			end = "?"
		}
		fmt.Fprintf(&b, "- %s usual %s-%s %s\n", c.Name, start, end, nanny.FormatWeekdays(c.DefaultWeekdays))
	}
	b.WriteString("Personal day off on a usual work day logs negative hours that cancel extra.\n")
	b.WriteString("Thai government paid holidays do not change hours owed unless they work that day (then pay the usual shift as extra).\n")
	return b.String()
}

func (r *Router) paidCalendar(ctx context.Context) []nanny.CalendarHoliday {
	items, err := r.Svc.ListPaidHolidays(ctx)
	if err != nil {
		return nil
	}
	out := make([]nanny.CalendarHoliday, 0, len(items))
	for _, h := range items {
		out = append(out, nanny.CalendarHoliday{Date: h.Date, Name: h.Name})
	}
	return out
}

func pickCaregiver(name string, cgs []models.Caregiver) *models.Caregiver {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "" || n == "nanny" || n == "the nanny" {
		if len(cgs) == 1 {
			return &cgs[0]
		}
		if len(cgs) > 0 && (n == "nanny" || n == "the nanny") {
			return &cgs[0]
		}
		return nil
	}
	for i := range cgs {
		if strings.EqualFold(cgs[i].Name, name) {
			return &cgs[i]
		}
	}
	return nil
}

func derefTime(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func bangkokNow() time.Time {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
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
