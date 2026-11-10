package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
)

type pantryListStore interface {
	EnsureShoppingList(ctx context.Context, householdID uuid.UUID) (models.List, error)
	CreateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error)
	GetListItemByPantryID(ctx context.Context, pantryID uuid.UUID) (*models.ListItem, error)
	UpdateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error)
	UpdatePantryStatus(ctx context.Context, id uuid.UUID, status string) error
}

func syncShoppingItem(ctx context.Context, store pantryListStore, householdID uuid.UUID, p models.PantryItem) error {
	existing, err := store.GetListItemByPantryID(ctx, p.ID)
	if err != nil {
		return err
	}
	if p.Status == models.PantryNeed {
		if existing == nil {
			list, err := store.EnsureShoppingList(ctx, householdID)
			if err != nil {
				return err
			}
			_, err = store.CreateListItem(ctx, models.ListItem{
				ListID:       list.ID,
				Title:        p.Name,
				Done:         false,
				PantryItemID: &p.ID,
			})
			return err
		}
		if existing.Done || existing.Title != p.Name {
			existing.Done = false
			existing.Title = p.Name
			_, err = store.UpdateListItem(ctx, *existing)
			return err
		}
		return nil
	}
	if existing != nil && !existing.Done {
		existing.Done = true
		_, err = store.UpdateListItem(ctx, *existing)
		return err
	}
	return nil
}

func applyListDoneToPantry(ctx context.Context, store pantryListStore, item models.ListItem) error {
	if item.PantryItemID == nil {
		return nil
	}
	status := models.PantryNeed
	if item.Done {
		status = models.PantryBought
	}
	return store.UpdatePantryStatus(ctx, *item.PantryItemID, status)
}
