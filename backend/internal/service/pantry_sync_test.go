package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePantryStore struct {
	household uuid.UUID
	pantry    map[uuid.UUID]models.PantryItem
	lists     map[uuid.UUID]models.List
	items     map[uuid.UUID]models.ListItem
}

func newFakePantryStore() *fakePantryStore {
	return &fakePantryStore{
		household: uuid.New(),
		pantry:    map[uuid.UUID]models.PantryItem{},
		lists:     map[uuid.UUID]models.List{},
		items:     map[uuid.UUID]models.ListItem{},
	}
}

func (f *fakePantryStore) EnsureShoppingList(_ context.Context, householdID uuid.UUID) (models.List, error) {
	for _, l := range f.lists {
		if l.Kind == models.ListKindShopping {
			return l, nil
		}
	}
	l := models.List{
		ID:          uuid.New(),
		HouseholdID: householdID,
		Title:       "Shopping",
		Kind:        models.ListKindShopping,
	}
	f.lists[l.ID] = l
	return l, nil
}

func (f *fakePantryStore) CreateListItem(_ context.Context, i models.ListItem) (models.ListItem, error) {
	i.ID = uuid.New()
	if i.PantryItemID != nil {
		if p, ok := f.pantry[*i.PantryItemID]; ok {
			i.NeededBy = p.NeededBy
		}
	}
	f.items[i.ID] = i
	return i, nil
}

func (f *fakePantryStore) GetListItemByPantryID(_ context.Context, pantryID uuid.UUID) (*models.ListItem, error) {
	for _, i := range f.items {
		if i.PantryItemID != nil && *i.PantryItemID == pantryID {
			cp := i
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakePantryStore) UpdateListItem(_ context.Context, i models.ListItem) (models.ListItem, error) {
	f.items[i.ID] = i
	return i, nil
}

func (f *fakePantryStore) UpdatePantryStatus(_ context.Context, id uuid.UUID, status string) error {
	p, ok := f.pantry[id]
	if !ok {
		return nil
	}
	p.Status = status
	f.pantry[id] = p
	return nil
}

func (f *fakePantryStore) visiblePantry() []models.PantryItem {
	out := make([]models.PantryItem, 0)
	for _, p := range f.pantry {
		if p.Status != models.PantryBought {
			out = append(out, p)
		}
	}
	return out
}

func (f *fakePantryStore) itemForPantry(pantryID uuid.UUID) models.ListItem {
	for _, i := range f.items {
		if i.PantryItemID != nil && *i.PantryItemID == pantryID {
			return i
		}
	}
	return models.ListItem{}
}

func TestNeedToBuySyncsShoppingList(t *testing.T) {
	ctx := context.Background()
	f := newFakePantryStore()
	p := models.PantryItem{
		ID:          uuid.New(),
		HouseholdID: f.household,
		Name:        "Milk",
		Status:      models.PantryNeed,
	}
	f.pantry[p.ID] = p

	require.NoError(t, syncShoppingItem(ctx, f, f.household, p))

	item := f.itemForPantry(p.ID)
	require.NotEqual(t, uuid.Nil, item.ID)
	assert.False(t, item.Done)
	assert.Equal(t, "Milk", item.Title)
	assert.Equal(t, "Shopping", f.lists[item.ListID].Title)
	assert.Len(t, f.visiblePantry(), 1)

	item.Done = true
	require.NoError(t, applyListDoneToPantry(ctx, f, item))
	assert.Equal(t, models.PantryBought, f.pantry[p.ID].Status)
	assert.Empty(t, f.visiblePantry())

	item.Done = false
	require.NoError(t, applyListDoneToPantry(ctx, f, item))
	assert.Equal(t, models.PantryNeed, f.pantry[p.ID].Status)
	assert.Len(t, f.visiblePantry(), 1)
}
