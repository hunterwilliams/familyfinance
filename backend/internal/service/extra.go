package service

import (
	"context"
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
	return s.repo.CreateCaregiver(ctx, c)
}

func (s *Service) CreateHourEntry(ctx context.Context, e models.CaregiverHourEntry) (models.CaregiverHourEntry, error) {
	h, err := s.EnsureHousehold(ctx)
	if err != nil {
		return models.CaregiverHourEntry{}, err
	}
	e.HouseholdID = h.ID
	if e.Hours == 0 && e.StartTime != nil && e.EndTime != nil {
		start, err1 := time.Parse("15:04", *e.StartTime)
		end, err2 := time.Parse("15:04", *e.EndTime)
		if err1 == nil && err2 == nil {
			e.Hours = nanny.HoursBetween(start, end)
		}
	}
	return s.repo.CreateHourEntry(ctx, e)
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
