package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetSettings(ctx context.Context, householdID uuid.UUID) (models.HouseholdSettings, error) {
	var s models.HouseholdSettings
	var openaiKey, lineSecret, lineToken *string
	err := r.pool.QueryRow(ctx, `
		SELECT openai_api_key, openai_model, line_channel_secret, line_channel_access_token
		FROM household_settings WHERE household_id = $1
	`, householdID).Scan(&openaiKey, &s.OpenAIModel, &lineSecret, &lineToken)
	if err == pgx.ErrNoRows {
		s.OpenAIModel = "gpt-4o"
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if openaiKey != nil {
		s.HasOpenAIKey = *openaiKey != ""
	}
	if lineSecret != nil {
		s.HasLineSecret = *lineSecret != ""
	}
	if lineToken != nil {
		s.HasLineToken = *lineToken != ""
	}
	return s, nil
}

func (r *Repository) GetSettingsSecrets(ctx context.Context, householdID uuid.UUID) (openaiKey, model, lineSecret, lineToken string, err error) {
	var k, m, s, t *string
	err = r.pool.QueryRow(ctx, `
		SELECT openai_api_key, openai_model, line_channel_secret, line_channel_access_token
		FROM household_settings WHERE household_id = $1
	`, householdID).Scan(&k, &m, &s, &t)
	if err == pgx.ErrNoRows {
		return "", "gpt-4o", "", "", nil
	}
	if err != nil {
		return "", "", "", "", err
	}
	if k != nil {
		openaiKey = *k
	}
	if m != nil {
		model = *m
	}
	if s != nil {
		lineSecret = *s
	}
	if t != nil {
		lineToken = *t
	}
	if model == "" {
		model = "gpt-4o"
	}
	return
}

func (r *Repository) UpsertSettings(ctx context.Context, householdID uuid.UUID, openaiKey, model, lineSecret, lineToken *string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO household_settings (household_id, openai_api_key, openai_model, line_channel_secret, line_channel_access_token)
		VALUES ($1, COALESCE($2,''), COALESCE($3,'gpt-4o'), COALESCE($4,''), COALESCE($5,''))
		ON CONFLICT (household_id) DO UPDATE SET
			openai_api_key = COALESCE($2, household_settings.openai_api_key),
			openai_model = COALESCE($3, household_settings.openai_model),
			line_channel_secret = COALESCE($4, household_settings.line_channel_secret),
			line_channel_access_token = COALESCE($5, household_settings.line_channel_access_token),
			updated_at = NOW()
	`, householdID, openaiKey, model, lineSecret, lineToken)
	return err
}

func (r *Repository) GetLineIdentity(ctx context.Context, lineUserID string) (*models.LineIdentity, error) {
	var id models.LineIdentity
	err := r.pool.QueryRow(ctx, `
		SELECT line_user_id, household_id, member_id, display_name
		FROM line_identities WHERE line_user_id = $1
	`, lineUserID).Scan(&id.LineUserID, &id.HouseholdID, &id.MemberID, &id.DisplayName)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *Repository) UpsertLineIdentity(ctx context.Context, id models.LineIdentity) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO line_identities (line_user_id, household_id, member_id, display_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (line_user_id) DO UPDATE SET
			member_id = COALESCE(EXCLUDED.member_id, line_identities.member_id),
			display_name = CASE WHEN EXCLUDED.display_name = '' THEN line_identities.display_name ELSE EXCLUDED.display_name END
	`, id.LineUserID, id.HouseholdID, id.MemberID, id.DisplayName)
	return err
}

func (r *Repository) CountLineIdentities(ctx context.Context, householdID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM line_identities WHERE household_id = $1`, householdID).Scan(&n)
	return n, err
}

func (r *Repository) BindLineMember(ctx context.Context, lineUserID string, memberID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE line_identities SET member_id = $2 WHERE line_user_id = $1`, lineUserID, memberID)
	return err
}

func (r *Repository) BindLineGroup(ctx context.Context, groupID string, householdID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO line_group_bindings (line_group_id, household_id) VALUES ($1, $2)
		ON CONFLICT (line_group_id) DO UPDATE SET household_id = EXCLUDED.household_id
	`, groupID, householdID)
	return err
}

func (r *Repository) GetLineGroup(ctx context.Context, groupID string) (uuid.UUID, error) {
	var hid uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT household_id FROM line_group_bindings WHERE line_group_id = $1`, groupID).Scan(&hid)
	if err == pgx.ErrNoRows {
		return uuid.Nil, nil
	}
	return hid, err
}

func (r *Repository) SavePending(ctx context.Context, lineUserID, chatID, kind string, payload any, expires time.Time) (uuid.UUID, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err = r.pool.QueryRow(ctx, `
		INSERT INTO pending_confirmations (line_user_id, chat_id, kind, payload, expires_at)
		VALUES ($1,$2,$3,$4,$5) RETURNING id
	`, lineUserID, chatID, kind, b, expires).Scan(&id)
	return id, err
}

func (r *Repository) LatestPending(ctx context.Context, lineUserID string) (id uuid.UUID, kind string, payload []byte, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT id, kind, payload FROM pending_confirmations
		WHERE line_user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1
	`, lineUserID).Scan(&id, &kind, &payload)
	if err == pgx.ErrNoRows {
		return uuid.Nil, "", nil, nil
	}
	return
}

func (r *Repository) DeletePending(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pending_confirmations WHERE id = $1`, id)
	return err
}

func (r *Repository) ListCaregivers(ctx context.Context, householdID uuid.UUID) ([]models.Caregiver, error) {
	rows, err := r.pool.Query(ctx, caregiverSelect+` FROM caregivers WHERE household_id = $1 ORDER BY name`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Caregiver, 0)
	for rows.Next() {
		c, err := scanCaregiver(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

const caregiverSelect = `
		SELECT id, household_id, name, hourly_rate_satang,
		       default_start_time::text, default_end_time::text, default_weekdays, active, created_at`

type caregiverRow interface {
	Scan(dest ...any) error
}

func scanCaregiver(row caregiverRow) (models.Caregiver, error) {
	var c models.Caregiver
	var days []int32
	err := row.Scan(&c.ID, &c.HouseholdID, &c.Name, &c.HourlyRateSatang, &c.DefaultStartTime, &c.DefaultEndTime, &days, &c.Active, &c.CreatedAt)
	if err != nil {
		return c, err
	}
	c.DefaultWeekdays = intSlice(days)
	normalizeCaregiverClocks(&c)
	return c, nil
}

func (r *Repository) GetCaregiver(ctx context.Context, householdID, id uuid.UUID) (*models.Caregiver, error) {
	c, err := scanCaregiver(r.pool.QueryRow(ctx, caregiverSelect+` FROM caregivers WHERE household_id = $1 AND id = $2`, householdID, id))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) CreateCaregiver(ctx context.Context, c models.Caregiver) (models.Caregiver, error) {
	c, err := scanCaregiver(r.pool.QueryRow(ctx, `
		INSERT INTO caregivers (household_id, name, hourly_rate_satang, default_start_time, default_end_time, default_weekdays, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, household_id, name, hourly_rate_satang,
		          default_start_time::text, default_end_time::text, default_weekdays, active, created_at
	`, c.HouseholdID, c.Name, c.HourlyRateSatang, c.DefaultStartTime, c.DefaultEndTime, int32Slice(c.DefaultWeekdays), c.Active))
	return c, err
}

func (r *Repository) UpdateCaregiver(ctx context.Context, c models.Caregiver) (models.Caregiver, error) {
	c, err := scanCaregiver(r.pool.QueryRow(ctx, `
		UPDATE caregivers
		SET name = $3, hourly_rate_satang = $4, default_start_time = $5, default_end_time = $6, default_weekdays = $7, active = $8
		WHERE household_id = $1 AND id = $2
		RETURNING id, household_id, name, hourly_rate_satang,
		          default_start_time::text, default_end_time::text, default_weekdays, active, created_at
	`, c.HouseholdID, c.ID, c.Name, c.HourlyRateSatang, c.DefaultStartTime, c.DefaultEndTime, int32Slice(c.DefaultWeekdays), c.Active))
	if err == pgx.ErrNoRows {
		return models.Caregiver{}, err
	}
	return c, err
}

func (r *Repository) DeleteCaregiver(ctx context.Context, householdID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM caregivers WHERE household_id = $1 AND id = $2`, householdID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func normalizeCaregiverClocks(c *models.Caregiver) {
	c.DefaultStartTime = clockPtr(c.DefaultStartTime)
	c.DefaultEndTime = clockPtr(c.DefaultEndTime)
}

func intSlice(in []int32) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

func int32Slice(in []int) []int32 {
	out := make([]int32, len(in))
	for i, v := range in {
		out[i] = int32(v)
	}
	return out
}

func clockPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			v := t.Format("15:04")
			return &v
		}
	}
	return &s
}

func (r *Repository) CreateHourEntry(ctx context.Context, e models.CaregiverHourEntry) (models.CaregiverHourEntry, error) {
	if e.Kind == "" {
		e.Kind = "extra"
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO caregiver_hour_entries (caregiver_id, household_id, work_date, start_time, end_time, hours, kind, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, caregiver_id, household_id, work_date, start_time::text, end_time::text, hours, kind, note, created_at
	`, e.CaregiverID, e.HouseholdID, e.WorkDate, e.StartTime, e.EndTime, e.Hours, e.Kind, e.Note).Scan(
		&e.ID, &e.CaregiverID, &e.HouseholdID, &e.WorkDate, &e.StartTime, &e.EndTime, &e.Hours, &e.Kind, &e.Note, &e.CreatedAt,
	)
	return e, err
}

func (r *Repository) ListHourEntries(ctx context.Context, householdID uuid.UUID, start, end time.Time) ([]models.CaregiverHourEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, caregiver_id, household_id, work_date, start_time::text, end_time::text, hours, kind, note, created_at
		FROM caregiver_hour_entries
		WHERE household_id = $1 AND work_date >= $2 AND work_date < $3
		ORDER BY work_date, created_at
	`, householdID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.CaregiverHourEntry, 0)
	for rows.Next() {
		var e models.CaregiverHourEntry
		if err := rows.Scan(&e.ID, &e.CaregiverID, &e.HouseholdID, &e.WorkDate, &e.StartTime, &e.EndTime, &e.Hours, &e.Kind, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) FindCaregiverByName(ctx context.Context, householdID uuid.UUID, name string) (*models.Caregiver, error) {
	c, err := scanCaregiver(r.pool.QueryRow(ctx, caregiverSelect+` FROM caregivers WHERE household_id = $1 AND LOWER(name) = LOWER($2) LIMIT 1`, householdID, name))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) CountPaidHolidays(ctx context.Context, householdID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM paid_holidays WHERE household_id = $1`, householdID).Scan(&n)
	return n, err
}

func (r *Repository) SeedPaidHolidays(ctx context.Context, householdID uuid.UUID, holidays []models.PaidHoliday) error {
	for _, h := range holidays {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO paid_holidays (household_id, holiday_date, name)
			VALUES ($1, $2, $3)
			ON CONFLICT (household_id, holiday_date) DO NOTHING
		`, householdID, h.Date, h.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) ListPaidHolidays(ctx context.Context, householdID uuid.UUID) ([]models.PaidHoliday, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, holiday_date::text, name, created_at
		FROM paid_holidays WHERE household_id = $1 ORDER BY holiday_date
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PaidHoliday, 0)
	for rows.Next() {
		var h models.PaidHoliday
		var rawDate string
		if err := rows.Scan(&h.ID, &h.HouseholdID, &rawDate, &h.Name, &h.CreatedAt); err != nil {
			return nil, err
		}
		h.Date = rawDate[:min(10, len(rawDate))]
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePaidHoliday(ctx context.Context, h models.PaidHoliday) (models.PaidHoliday, error) {
	var rawDate string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO paid_holidays (household_id, holiday_date, name)
		VALUES ($1, $2, $3)
		RETURNING id, household_id, holiday_date::text, name, created_at
	`, h.HouseholdID, h.Date, h.Name).Scan(&h.ID, &h.HouseholdID, &rawDate, &h.Name, &h.CreatedAt)
	if err != nil {
		return models.PaidHoliday{}, err
	}
	h.Date = rawDate[:min(10, len(rawDate))]
	return h, nil
}

func (r *Repository) DeletePaidHoliday(ctx context.Context, householdID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM paid_holidays WHERE household_id = $1 AND id = $2`, householdID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

const listItemSelect = `
	SELECT i.id, i.list_id, i.title, i.done, i.sort_order, i.created_at, i.pantry_item_id, p.needed_by
	FROM list_items i
	LEFT JOIN pantry_items p ON p.id = i.pantry_item_id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanListItem(row scanner) (models.ListItem, error) {
	var i models.ListItem
	err := row.Scan(&i.ID, &i.ListID, &i.Title, &i.Done, &i.SortOrder, &i.CreatedAt, &i.PantryItemID, &i.NeededBy)
	return i, err
}

func scanPantry(row scanner) (models.PantryItem, error) {
	var p models.PantryItem
	err := row.Scan(&p.ID, &p.HouseholdID, &p.Name, &p.Status, &p.ExpiresOn, &p.NeededBy, &p.Notes, &p.RebuyHint, &p.CreatedAt)
	return p, err
}

func (r *Repository) ListLists(ctx context.Context, householdID uuid.UUID) ([]models.List, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, title, kind, created_at
		FROM lists WHERE household_id = $1
		ORDER BY CASE WHEN kind = 'shopping' THEN 0 ELSE 1 END, created_at
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.List, 0)
	for rows.Next() {
		var l models.List
		if err := rows.Scan(&l.ID, &l.HouseholdID, &l.Title, &l.Kind, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) CreateList(ctx context.Context, l models.List) (models.List, error) {
	if l.Kind == "" {
		l.Kind = models.ListKindCustom
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO lists (household_id, title, kind) VALUES ($1,$2,$3)
		RETURNING id, household_id, title, kind, created_at
	`, l.HouseholdID, l.Title, l.Kind).Scan(&l.ID, &l.HouseholdID, &l.Title, &l.Kind, &l.CreatedAt)
	return l, err
}

func (r *Repository) EnsureShoppingList(ctx context.Context, householdID uuid.UUID) (models.List, error) {
	var l models.List
	err := r.pool.QueryRow(ctx, `
		INSERT INTO lists (household_id, title, kind)
		VALUES ($1, 'Shopping', 'shopping')
		ON CONFLICT (household_id) WHERE kind = 'shopping'
		DO UPDATE SET title = lists.title
		RETURNING id, household_id, title, kind, created_at
	`, householdID).Scan(&l.ID, &l.HouseholdID, &l.Title, &l.Kind, &l.CreatedAt)
	return l, err
}

func (r *Repository) ListItems(ctx context.Context, listID uuid.UUID) ([]models.ListItem, error) {
	rows, err := r.pool.Query(ctx, listItemSelect+` WHERE i.list_id = $1 ORDER BY i.sort_order, i.created_at`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ListItem, 0)
	for rows.Next() {
		i, err := scanListItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) GetListItem(ctx context.Context, id uuid.UUID) (models.ListItem, error) {
	return scanListItem(r.pool.QueryRow(ctx, listItemSelect+` WHERE i.id = $1`, id))
}

func (r *Repository) GetListItemByPantryID(ctx context.Context, pantryID uuid.UUID) (*models.ListItem, error) {
	i, err := scanListItem(r.pool.QueryRow(ctx, listItemSelect+` WHERE i.pantry_item_id = $1`, pantryID))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *Repository) CreateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error) {
	return scanListItem(r.pool.QueryRow(ctx, `
		INSERT INTO list_items (list_id, title, done, sort_order, pantry_item_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, list_id, title, done, sort_order, created_at, pantry_item_id,
			(SELECT needed_by FROM pantry_items WHERE id = $5)
	`, i.ListID, i.Title, i.Done, i.SortOrder, i.PantryItemID))
}

func (r *Repository) UpdateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error) {
	return scanListItem(r.pool.QueryRow(ctx, `
		UPDATE list_items SET title=$2, done=$3 WHERE id=$1
		RETURNING id, list_id, title, done, sort_order, created_at, pantry_item_id,
			(SELECT needed_by FROM pantry_items WHERE id = list_items.pantry_item_id)
	`, i.ID, i.Title, i.Done))
}

func (r *Repository) ListPantry(ctx context.Context, householdID uuid.UUID) ([]models.PantryItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, status, expires_on, needed_by, notes, rebuy_hint, created_at
		FROM pantry_items
		WHERE household_id = $1 AND status <> 'bought'
		ORDER BY CASE WHEN status = 'need' THEN 1 ELSE 0 END,
			expires_on NULLS LAST, needed_by NULLS LAST, name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PantryItem, 0)
	for rows.Next() {
		p, err := scanPantry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetPantry(ctx context.Context, householdID, id uuid.UUID) (models.PantryItem, error) {
	return scanPantry(r.pool.QueryRow(ctx, `
		SELECT id, household_id, name, status, expires_on, needed_by, notes, rebuy_hint, created_at
		FROM pantry_items WHERE household_id = $1 AND id = $2
	`, householdID, id))
}

func (r *Repository) CreatePantry(ctx context.Context, p models.PantryItem) (models.PantryItem, error) {
	if p.Status == "" {
		p.Status = models.PantryOnHand
	}
	return scanPantry(r.pool.QueryRow(ctx, `
		INSERT INTO pantry_items (household_id, name, status, expires_on, needed_by, notes, rebuy_hint)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, household_id, name, status, expires_on, needed_by, notes, rebuy_hint, created_at
	`, p.HouseholdID, p.Name, p.Status, p.ExpiresOn, p.NeededBy, p.Notes, p.RebuyHint))
}

func (r *Repository) UpdatePantry(ctx context.Context, p models.PantryItem) (models.PantryItem, error) {
	return scanPantry(r.pool.QueryRow(ctx, `
		UPDATE pantry_items
		SET name=$3, status=$4, expires_on=$5, needed_by=$6, notes=$7, rebuy_hint=$8
		WHERE household_id=$1 AND id=$2
		RETURNING id, household_id, name, status, expires_on, needed_by, notes, rebuy_hint, created_at
	`, p.HouseholdID, p.ID, p.Name, p.Status, p.ExpiresOn, p.NeededBy, p.Notes, p.RebuyHint))
}

func (r *Repository) UpdatePantryStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE pantry_items SET status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
