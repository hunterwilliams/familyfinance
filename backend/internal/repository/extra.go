package repository

import (
	"context"
	"encoding/json"
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, hourly_rate_satang, active, created_at
		FROM caregivers WHERE household_id = $1 ORDER BY name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Caregiver, 0)
	for rows.Next() {
		var c models.Caregiver
		if err := rows.Scan(&c.ID, &c.HouseholdID, &c.Name, &c.HourlyRateSatang, &c.Active, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCaregiver(ctx context.Context, c models.Caregiver) (models.Caregiver, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO caregivers (household_id, name, hourly_rate_satang, active)
		VALUES ($1,$2,$3,$4)
		RETURNING id, household_id, name, hourly_rate_satang, active, created_at
	`, c.HouseholdID, c.Name, c.HourlyRateSatang, c.Active).Scan(
		&c.ID, &c.HouseholdID, &c.Name, &c.HourlyRateSatang, &c.Active, &c.CreatedAt,
	)
	return c, err
}

func (r *Repository) CreateHourEntry(ctx context.Context, e models.CaregiverHourEntry) (models.CaregiverHourEntry, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO caregiver_hour_entries (caregiver_id, household_id, work_date, start_time, end_time, hours, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, caregiver_id, household_id, work_date, start_time::text, end_time::text, hours, note, created_at
	`, e.CaregiverID, e.HouseholdID, e.WorkDate, e.StartTime, e.EndTime, e.Hours, e.Note).Scan(
		&e.ID, &e.CaregiverID, &e.HouseholdID, &e.WorkDate, &e.StartTime, &e.EndTime, &e.Hours, &e.Note, &e.CreatedAt,
	)
	return e, err
}

func (r *Repository) ListHourEntries(ctx context.Context, householdID uuid.UUID, start, end time.Time) ([]models.CaregiverHourEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, caregiver_id, household_id, work_date, start_time::text, end_time::text, hours, note, created_at
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
		if err := rows.Scan(&e.ID, &e.CaregiverID, &e.HouseholdID, &e.WorkDate, &e.StartTime, &e.EndTime, &e.Hours, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) FindCaregiverByName(ctx context.Context, householdID uuid.UUID, name string) (*models.Caregiver, error) {
	var c models.Caregiver
	err := r.pool.QueryRow(ctx, `
		SELECT id, household_id, name, hourly_rate_satang, active, created_at
		FROM caregivers WHERE household_id = $1 AND LOWER(name) = LOWER($2) LIMIT 1
	`, householdID, name).Scan(&c.ID, &c.HouseholdID, &c.Name, &c.HourlyRateSatang, &c.Active, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListLists(ctx context.Context, householdID uuid.UUID) ([]models.List, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, household_id, title, created_at FROM lists WHERE household_id = $1 ORDER BY created_at`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.List, 0)
	for rows.Next() {
		var l models.List
		if err := rows.Scan(&l.ID, &l.HouseholdID, &l.Title, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) CreateList(ctx context.Context, l models.List) (models.List, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO lists (household_id, title) VALUES ($1,$2)
		RETURNING id, household_id, title, created_at
	`, l.HouseholdID, l.Title).Scan(&l.ID, &l.HouseholdID, &l.Title, &l.CreatedAt)
	return l, err
}

func (r *Repository) ListItems(ctx context.Context, listID uuid.UUID) ([]models.ListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, list_id, title, done, sort_order, created_at FROM list_items WHERE list_id = $1 ORDER BY sort_order, created_at
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ListItem, 0)
	for rows.Next() {
		var i models.ListItem
		if err := rows.Scan(&i.ID, &i.ListID, &i.Title, &i.Done, &i.SortOrder, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) CreateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO list_items (list_id, title, done, sort_order) VALUES ($1,$2,$3,$4)
		RETURNING id, list_id, title, done, sort_order, created_at
	`, i.ListID, i.Title, i.Done, i.SortOrder).Scan(&i.ID, &i.ListID, &i.Title, &i.Done, &i.SortOrder, &i.CreatedAt)
	return i, err
}

func (r *Repository) UpdateListItem(ctx context.Context, i models.ListItem) (models.ListItem, error) {
	err := r.pool.QueryRow(ctx, `
		UPDATE list_items SET title=$2, done=$3 WHERE id=$1
		RETURNING id, list_id, title, done, sort_order, created_at
	`, i.ID, i.Title, i.Done).Scan(&i.ID, &i.ListID, &i.Title, &i.Done, &i.SortOrder, &i.CreatedAt)
	return i, err
}

func (r *Repository) ListPantry(ctx context.Context, householdID uuid.UUID) ([]models.PantryItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, household_id, name, expires_on, notes, rebuy_hint, created_at
		FROM pantry_items WHERE household_id = $1 ORDER BY expires_on NULLS LAST, name
	`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PantryItem, 0)
	for rows.Next() {
		var p models.PantryItem
		if err := rows.Scan(&p.ID, &p.HouseholdID, &p.Name, &p.ExpiresOn, &p.Notes, &p.RebuyHint, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePantry(ctx context.Context, p models.PantryItem) (models.PantryItem, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO pantry_items (household_id, name, expires_on, notes, rebuy_hint)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, household_id, name, expires_on, notes, rebuy_hint, created_at
	`, p.HouseholdID, p.Name, p.ExpiresOn, p.Notes, p.RebuyHint).Scan(
		&p.ID, &p.HouseholdID, &p.Name, &p.ExpiresOn, &p.Notes, &p.RebuyHint, &p.CreatedAt,
	)
	return p, err
}
