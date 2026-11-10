package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/domain/nanny"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/hunterwilliams/familyfinance/internal/repository"
)

func (s *Service) Repo() *repository.Repository {
	return s.repo
}

func (s *Service) GetSettings(ctx context.Context) (models.HouseholdSettings, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.HouseholdSettings{}, err
	}
	return s.repo.GetSettings(ctx, h.ID)
}

func (s *Service) UpdateSettings(ctx context.Context, openaiKey, model, lineSecret, lineToken *string) (models.HouseholdSettings, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.HouseholdSettings{}, err
	}
	if err := s.repo.UpsertSettings(ctx, h.ID, openaiKey, model, lineSecret, lineToken); err != nil {
		return models.HouseholdSettings{}, err
	}
	return s.repo.GetSettings(ctx, h.ID)
}

func (s *Service) ListCaregivers(ctx context.Context) ([]models.Caregiver, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCaregivers(ctx, h.ID)
}

func (s *Service) CreateCaregiver(ctx context.Context, c models.Caregiver) (models.Caregiver, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Caregiver{}, err
	}
	c.HouseholdID = h.ID
	if !c.Active {
		c.Active = true
	}
	normalizeCaregiverTimes(&c)
	c.DefaultWeekdays = nanny.NormalizeWeekdays(c.DefaultWeekdays)
	return s.repo.CreateCaregiver(ctx, c)
}

func (s *Service) UpdateCaregiver(ctx context.Context, c models.Caregiver) (models.Caregiver, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.Caregiver{}, err
	}
	c.HouseholdID = h.ID
	if !c.Active {
		c.Active = true
	}
	normalizeCaregiverTimes(&c)
	if c.DefaultWeekdays == nil {
		existing, err := s.repo.GetCaregiver(ctx, h.ID, c.ID)
		if err != nil {
			return models.Caregiver{}, err
		}
		if existing != nil {
			c.DefaultWeekdays = existing.DefaultWeekdays
		}
	}
	c.DefaultWeekdays = nanny.NormalizeWeekdays(c.DefaultWeekdays)
	return s.repo.UpdateCaregiver(ctx, c)
}

func normalizeCaregiverTimes(c *models.Caregiver) {
	c.DefaultStartTime = blankTimeNil(c.DefaultStartTime)
	c.DefaultEndTime = blankTimeNil(c.DefaultEndTime)
}

func blankTimeNil(p *string) *string {
	if p == nil {
		return nil
	}
	s := nanny.ClockHHMM(*p)
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) CreateHourEntry(ctx context.Context, e models.CaregiverHourEntry) (models.CaregiverHourEntry, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.CaregiverHourEntry{}, err
	}
	e.HouseholdID = h.ID
	paid, err := s.paidCalendar(ctx, h.ID)
	if err != nil {
		return models.CaregiverHourEntry{}, err
	}
	if strings.EqualFold(e.Kind, nanny.KindHoliday) {
		cg, err := s.repo.GetCaregiver(ctx, h.ID, e.CaregiverID)
		if err != nil {
			return models.CaregiverHourEntry{}, err
		}
		if cg == nil {
			return models.CaregiverHourEntry{}, fmt.Errorf("caregiver not found")
		}
		intent := nanny.HolidayFor(caregiverRef(*cg), e.WorkDate, paid)
		if intent.Kind == nanny.KindPaidHoliday {
			return models.CaregiverHourEntry{}, fmt.Errorf("%s is a paid holiday (%s) and does not affect hours owed unless they work", e.WorkDate.Format("2006-01-02"), intent.Note)
		}
		if intent.Hours == 0 {
			return models.CaregiverHourEntry{}, fmt.Errorf("%s does not usually work on %s (%s)", cg.Name, e.WorkDate.Format("2006-01-02"), nanny.FormatWeekdays(cg.DefaultWeekdays))
		}
		e.Hours = intent.Hours
		e.StartTime = strPtr(intent.StartTime)
		e.EndTime = strPtr(intent.EndTime)
		if strings.TrimSpace(e.Note) == "" {
			e.Note = "holiday"
		}
		e.Kind = nanny.KindHoliday
	} else if strings.EqualFold(e.Kind, nanny.KindWorkedHoliday) {
		cg, err := s.repo.GetCaregiver(ctx, h.ID, e.CaregiverID)
		if err != nil {
			return models.CaregiverHourEntry{}, err
		}
		if cg == nil {
			return models.CaregiverHourEntry{}, fmt.Errorf("caregiver not found")
		}
		intent := nanny.WorkedHolidayFor(caregiverRef(*cg), e.WorkDate, paid)
		if intent.Hours == 0 {
			return models.CaregiverHourEntry{}, fmt.Errorf("%s is not a paid holiday. Add it to the paid holiday list first", e.WorkDate.Format("2006-01-02"))
		}
		e.Hours = intent.Hours
		e.StartTime = strPtr(intent.StartTime)
		e.EndTime = strPtr(intent.EndTime)
		if strings.TrimSpace(e.Note) == "" {
			e.Note = intent.Note
		}
		e.Kind = nanny.KindWorkedHoliday
	} else {
		if e.Kind == "" {
			e.Kind = nanny.KindExtra
		}
		if e.Hours == 0 && e.StartTime != nil && e.EndTime != nil {
			e.Hours = nanny.HoursBetweenClocks(*e.StartTime, *e.EndTime)
		}
	}
	return s.repo.CreateHourEntry(ctx, e)
}

func (s *Service) ListPaidHolidays(ctx context.Context) ([]models.PaidHoliday, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePaidHolidays(ctx, h.ID); err != nil {
		return nil, err
	}
	return s.repo.ListPaidHolidays(ctx, h.ID)
}

func (s *Service) CreatePaidHoliday(ctx context.Context, date, name string) (models.PaidHoliday, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.PaidHoliday{}, err
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return models.PaidHoliday{}, fmt.Errorf("date must be YYYY-MM-DD")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Paid holiday"
	}
	item, err := s.repo.CreatePaidHoliday(ctx, models.PaidHoliday{HouseholdID: h.ID, Date: date, Name: name})
	if err != nil {
		if strings.Contains(err.Error(), "paid_holidays") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return models.PaidHoliday{}, fmt.Errorf("that date is already on the paid holiday list")
		}
		return models.PaidHoliday{}, err
	}
	return item, nil
}

func (s *Service) DeletePaidHoliday(ctx context.Context, id uuid.UUID) error {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeletePaidHoliday(ctx, h.ID, id)
}

func (s *Service) ensurePaidHolidays(ctx context.Context, householdID uuid.UUID) error {
	n, err := s.repo.CountPaidHolidays(ctx, householdID)
	if err != nil || n > 0 {
		return err
	}
	defs := nanny.DefaultThaiHolidays()
	seed := make([]models.PaidHoliday, 0, len(defs))
	for _, d := range defs {
		seed = append(seed, models.PaidHoliday{Date: d.Date, Name: d.Name})
	}
	return s.repo.SeedPaidHolidays(ctx, householdID, seed)
}

func (s *Service) paidCalendar(ctx context.Context, householdID uuid.UUID) ([]nanny.CalendarHoliday, error) {
	if err := s.ensurePaidHolidays(ctx, householdID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListPaidHolidays(ctx, householdID)
	if err != nil {
		return nil, err
	}
	out := make([]nanny.CalendarHoliday, 0, len(items))
	for _, h := range items {
		out = append(out, nanny.CalendarHoliday{Date: h.Date, Name: h.Name})
	}
	return out, nil
}

func caregiverRef(c models.Caregiver) nanny.Ref {
	start, end := "", ""
	if c.DefaultStartTime != nil {
		start = *c.DefaultStartTime
	}
	if c.DefaultEndTime != nil {
		end = *c.DefaultEndTime
	}
	return nanny.Ref{
		Name:         c.Name,
		DefaultStart: start,
		DefaultEnd:   end,
		DefaultDays:  c.DefaultWeekdays,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) ListHourEntries(ctx context.Context, month string) ([]models.CaregiverHourEntry, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	start, err := repository.ParseMonth(month)
	if err != nil {
		return nil, err
	}
	return s.repo.ListHourEntries(ctx, h.ID, start, start.AddDate(0, 1, 0))
}

func (s *Service) ListLists(ctx context.Context) ([]models.List, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	lists, err := s.repo.ListLists(ctx, h.ID)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		items, err := s.repo.ListItems(ctx, lists[i].ID)
		if err != nil {
			return nil, err
		}
		lists[i].Items = items
	}
	return lists, nil
}

func (s *Service) CreateList(ctx context.Context, title string) (models.List, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.List{}, err
	}
	return s.repo.CreateList(ctx, models.List{HouseholdID: h.ID, Title: title})
}

func (s *Service) CreateListItem(ctx context.Context, listID uuid.UUID, title string) (models.ListItem, error) {
	return s.repo.CreateListItem(ctx, models.ListItem{ListID: listID, Title: title})
}

func (s *Service) UpdateListItem(ctx context.Context, item models.ListItem) (models.ListItem, error) {
	return s.repo.UpdateListItem(ctx, item)
}

func (s *Service) ListPantry(ctx context.Context) ([]models.PantryItem, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListPantry(ctx, h.ID)
}

func (s *Service) CreatePantry(ctx context.Context, p models.PantryItem) (models.PantryItem, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.PantryItem{}, err
	}
	p.HouseholdID = h.ID
	return s.repo.CreatePantry(ctx, p)
}
