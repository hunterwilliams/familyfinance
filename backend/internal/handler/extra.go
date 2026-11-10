package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/ai"
	"github.com/hunterwilliams/familyfinance/internal/domain/namematch"
	"github.com/hunterwilliams/familyfinance/internal/linebot"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/rs/zerolog/log"
)

func (h *Handler) getSettings(c *gin.Context) {
	s, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.cfg.OpenAIAPIKey != "" {
		s.HasOpenAIKey = true
	}
	if h.cfg.LineChannelSecret != "" {
		s.HasLineSecret = true
	}
	if h.cfg.LineChannelAccessToken != "" {
		s.HasLineToken = true
	}
	c.JSON(http.StatusOK, s)
}

func omitBlank(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func (h *Handler) patchSettings(c *gin.Context) {
	var req struct {
		OpenAIAPIKey           *string `json:"openai_api_key"`
		OpenAIModel            *string `json:"openai_model"`
		LineChannelSecret      *string `json:"line_channel_secret"`
		LineChannelAccessToken *string `json:"line_channel_access_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.OpenAIModel = omitBlank(req.OpenAIModel)
	req.LineChannelSecret = omitBlank(req.LineChannelSecret)
	req.LineChannelAccessToken = omitBlank(req.LineChannelAccessToken)
	if req.OpenAIAPIKey != nil {
		n := ai.NormalizeKey(*req.OpenAIAPIKey)
		if n == "" {
			req.OpenAIAPIKey = nil
		} else {
			req.OpenAIAPIKey = &n
			model := ""
			if req.OpenAIModel != nil {
				model = *req.OpenAIModel
			}
			client := ai.NewWithBase(n, model, h.cfg.OpenAIBaseURL)
			if err := client.Validate(c.Request.Context()); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}
	s, err := h.svc.UpdateSettings(c.Request.Context(), req.OpenAIAPIKey, req.OpenAIModel, req.LineChannelSecret, req.LineChannelAccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) testAI(c *gin.Context) {
	var req struct {
		OpenAIAPIKey *string `json:"openai_api_key"`
		OpenAIModel  *string `json:"openai_model"`
		Check        string  `json:"check"`
	}
	_ = c.ShouldBindJSON(&req)
	key, model := h.openaiCreds(c)
	if k := omitBlank(req.OpenAIAPIKey); k != nil {
		key = ai.NormalizeKey(*k)
	}
	if m := omitBlank(req.OpenAIModel); m != nil {
		model = *m
	}
	client := h.openaiClientFrom(key, model)
	if req.Check == "model" {
		if err := client.CheckModel(c.Request.Context()); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error(), "model": client.Model})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "model": client.Model, "message": "Model " + client.Model + " is available"})
		return
	}
	if err := client.Validate(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "model": client.Model, "message": "API key works"})
}

func (h *Handler) testLine(c *gin.Context) {
	var req struct {
		LineChannelAccessToken *string `json:"line_channel_access_token"`
	}
	_ = c.ShouldBindJSON(&req)
	_, token := h.lineCreds(c)
	if t := omitBlank(req.LineChannelAccessToken); t != nil {
		token = *t
	}
	client := linebot.NewClient(token)
	info, err := client.GetBotInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "error": err.Error()})
		return
	}
	out := gin.H{
		"ok":        true,
		"bot_name":  info.DisplayName,
		"bot_id":    info.BasicID,
		"chat_mode": info.ChatMode,
	}
	if n, ferr := client.CountFollowers(c.Request.Context()); ferr == nil {
		out["friend_count"] = n
	}
	if hh, herr := h.svc.EnsureHousehold(c.Request.Context()); herr == nil {
		if n, cerr := h.svc.Repo().CountLineIdentities(c.Request.Context(), hh.ID); cerr == nil {
			out["linked_users"] = n
		}
	}
	msg := "LINE messaging token works"
	if info.DisplayName != "" {
		msg += " for " + info.DisplayName
	}
	if n, ok := out["friend_count"].(int); ok {
		msg += fmt.Sprintf(". %d friend(s) on the official account", n)
	}
	if n, ok := out["linked_users"].(int); ok {
		msg += fmt.Sprintf(". %d user(s) have messaged this bot", n)
	}
	out["message"] = msg
	c.JSON(http.StatusOK, out)
}

func (h *Handler) openaiCreds(c *gin.Context) (key, model string) {
	key = ai.NormalizeKey(h.cfg.OpenAIAPIKey)
	model = strings.TrimSpace(h.cfg.OpenAIModel)
	hh, err := h.svc.EnsureHousehold(c.Request.Context())
	if err == nil {
		k, m, _, _, _ := h.svc.Repo().GetSettingsSecrets(c.Request.Context(), hh.ID)
		k = ai.NormalizeKey(k)
		if k != "" {
			key = k
		}
		if strings.TrimSpace(m) != "" {
			model = m
		}
	}
	return key, ai.DefaultModel(key, model)
}

func (h *Handler) openaiClientFrom(key, model string) *ai.Client {
	return ai.NewWithBase(key, model, h.cfg.OpenAIBaseURL)
}

func (h *Handler) openaiClient(c *gin.Context) *ai.Client {
	key, model := h.openaiCreds(c)
	return h.openaiClientFrom(key, model)
}

func (h *Handler) lineCreds(c *gin.Context) (secret, token string) {
	secret = h.cfg.LineChannelSecret
	token = h.cfg.LineChannelAccessToken
	hh, err := h.svc.EnsureHousehold(c.Request.Context())
	if err == nil {
		_, _, s, t, _ := h.svc.Repo().GetSettingsSecrets(c.Request.Context(), hh.ID)
		if secret == "" {
			secret = s
		}
		if token == "" {
			token = t
		}
	}
	return
}

func (h *Handler) parseReceipt(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
	file, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || strings.Contains(strings.ToLower(err.Error()), "too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Photo is too large. Use an image under 2 MB, or a smaller JPEG."})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	members, _ := h.svc.ListMembers(c.Request.Context())
	client := h.openaiClient(c)
	fields, err := client.ParseReceiptImage(c.Request.Context(), base64.StdEncoding.EncodeToString(b), file.Header.Get("Content-Type"), memberHint(members))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var amount *int64
	if fields.AmountTHB > 0 {
		v := int64(fields.AmountTHB * 100)
		amount = &v
	}
	var date *string
	if fields.Date != "" {
		date = &fields.Date
	}
	var merchant *string
	if fields.Merchant != "" {
		merchant = &fields.Merchant
	}
	texts := append([]string{fields.AccountHolder}, fields.NamesVisible...)
	hit := namematch.MatchAll(texts, members)
	log.Info().
		Str("account_holder", fields.AccountHolder).
		Strs("names_visible", fields.NamesVisible).
		Bool("is_joint", hit.IsJoint).
		Interface("paid_by", hit.MemberID).
		Msg("receipt payer match")
	out := models.ReceiptParse{
		AmountSatang:   amount,
		Date:           date,
		Merchant:       merchant,
		AccountHolder:  fields.AccountHolder,
		PaidByMemberID: hit.MemberID,
		IsJoint:        hit.IsJoint,
	}
	c.JSON(http.StatusOK, out)
}

func memberHint(members []models.Member) string {
	parts := make([]string, 0, len(members))
	for _, m := range members {
		label := strings.TrimSpace(m.Name)
		if th := strings.TrimSpace(m.NameTh); th != "" {
			label += " / " + th
		}
		if label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, "; ")
}

func (h *Handler) listCaregivers(c *gin.Context) {
	items, err := h.svc.ListCaregivers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createCaregiver(c *gin.Context) {
	var req models.Caregiver
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreateCaregiver(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateCaregiver(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.Caregiver
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	item, err := h.svc.UpdateCaregiver(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) listCaregiverHours(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	items, err := h.svc.ListHourEntries(c.Request.Context(), month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createCaregiverHours(c *gin.Context) {
	var req struct {
		CaregiverID uuid.UUID `json:"caregiver_id" binding:"required"`
		WorkDate    string    `json:"work_date" binding:"required"`
		StartTime   *string   `json:"start_time"`
		EndTime     *string   `json:"end_time"`
		Hours       float64   `json:"hours"`
		Kind        string    `json:"kind"`
		Note        string    `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := time.Parse("2006-01-02", req.WorkDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "work_date must be YYYY-MM-DD"})
		return
	}
	item, err := h.svc.CreateHourEntry(c.Request.Context(), models.CaregiverHourEntry{
		CaregiverID: req.CaregiverID,
		WorkDate:    d,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Hours:       req.Hours,
		Kind:        req.Kind,
		Note:        req.Note,
	})
	if err != nil {
		if strings.Contains(err.Error(), "does not usually work") || strings.Contains(err.Error(), "paid holiday") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) listPaidHolidays(c *gin.Context) {
	items, err := h.svc.ListPaidHolidays(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createPaidHoliday(c *gin.Context) {
	var req struct {
		Date string `json:"date" binding:"required"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreatePaidHoliday(c.Request.Context(), req.Date, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already") || strings.Contains(err.Error(), "YYYY-MM-DD") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) deletePaidHoliday(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeletePaidHoliday(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listLists(c *gin.Context) {
	items, err := h.svc.ListLists(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createList(c *gin.Context) {
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreateList(c.Request.Context(), req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) createListItem(c *gin.Context) {
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreateListItem(c.Request.Context(), listID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updateListItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Title string `json:"title"`
		Done  bool   `json:"done"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.UpdateListItem(c.Request.Context(), models.ListItem{ID: id, Title: req.Title, Done: req.Done})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) listPantry(c *gin.Context) {
	items, err := h.svc.ListPantry(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func parseOptionalDate(raw *string, field string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	d, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be YYYY-MM-DD", field)
	}
	return &d, nil
}

func (h *Handler) createPantry(c *gin.Context) {
	var req struct {
		Name      string  `json:"name" binding:"required"`
		Status    string  `json:"status"`
		ExpiresOn *string `json:"expires_on"`
		NeededBy  *string `json:"needed_by"`
		Notes     string  `json:"notes"`
		RebuyHint string  `json:"rebuy_hint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := models.PantryItem{Name: req.Name, Status: req.Status, Notes: req.Notes, RebuyHint: req.RebuyHint}
	var err error
	p.ExpiresOn, err = parseOptionalDate(req.ExpiresOn, "expires_on")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.NeededBy, err = parseOptionalDate(req.NeededBy, "needed_by")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreatePantry(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) updatePantry(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name      string  `json:"name"`
		Status    string  `json:"status"`
		ExpiresOn *string `json:"expires_on"`
		NeededBy  *string `json:"needed_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := models.PantryItem{ID: id, Name: req.Name, Status: req.Status}
	p.ExpiresOn, err = parseOptionalDate(req.ExpiresOn, "expires_on")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.NeededBy, err = parseOptionalDate(req.NeededBy, "needed_by")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.UpdatePantry(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}
