package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/auth"
	"github.com/hunterwilliams/familyfinance/internal/config"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/hunterwilliams/familyfinance/internal/service"
)

type Handler struct {
	svc         *service.Service
	cfg         config.Config
}

func New(svc *service.Service, cfg config.Config) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)
	r.POST("/webhooks/line", h.lineWebhook)

	authg := r.Group("/api/v1/auth")
	{
		authg.POST("/dev-login", h.devLogin)
		authg.GET("/line/start", h.lineLoginStart)
		authg.GET("/line/callback", h.lineLoginCallback)
		authg.GET("/me", h.authMiddleware(), h.authMe)
		authg.POST("/link-member", h.authMiddleware(), h.linkMember)
	}

	v1 := r.Group("/api/v1")
	v1.Use(h.authMiddleware())
	{
		v1.GET("/members", h.listMembers)
		v1.POST("/members", h.createMember)
		v1.PATCH("/members/:id", h.updateMember)

		v1.GET("/accounts", h.listAccounts)
		v1.POST("/accounts", h.createAccount)

		v1.GET("/categories", h.listCategories)
		v1.POST("/categories", h.createCategory)

		v1.GET("/transactions", h.listTransactions)
		v1.POST("/transactions", h.createTransaction)
		v1.PATCH("/transactions/bulk", h.bulkUpdateTransactions)

		v1.GET("/balance-snapshots", h.listBalanceSnapshots)
		v1.POST("/balance-snapshots", h.createBalanceSnapshot)

		v1.GET("/cc-balances", h.listCCBalances)
		v1.POST("/cc-balances", h.createCCBalance)

		v1.GET("/settlements", h.listSettlements)
		v1.POST("/settlements", h.createSettlement)

		v1.GET("/reports/settlement", h.settlementReport)
		v1.GET("/reports/effective-balance", h.effectiveBalance)

		v1.GET("/savings-goals", h.listSavingsGoals)
		v1.POST("/savings-goals", h.createSavingsGoal)
		v1.PATCH("/savings-goals/:id", h.updateSavingsGoal)

		v1.POST("/receipts", h.uploadReceipt)
		v1.POST("/receipts/parse", h.parseReceipt)

		v1.GET("/settings", h.getSettings)
		v1.PATCH("/settings", h.patchSettings)

		v1.GET("/caregivers", h.listCaregivers)
		v1.POST("/caregivers", h.createCaregiver)
		v1.GET("/caregiver-hours", h.listCaregiverHours)
		v1.POST("/caregiver-hours", h.createCaregiverHours)

		v1.GET("/lists", h.listLists)
		v1.POST("/lists", h.createList)
		v1.POST("/lists/:id/items", h.createListItem)
		v1.PATCH("/list-items/:id", h.updateListItem)

		v1.GET("/pantry-items", h.listPantry)
		v1.POST("/pantry-items", h.createPantry)
	}
}

func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.cfg.SkipAuth {
			c.Next()
			return
		}
		if h.cfg.APIKey != "" {
			key := c.GetHeader("X-API-Key")
			if key == "" {
				key = c.Query("api_key")
			}
			if key == h.cfg.APIKey {
				c.Next()
				return
			}
		}
		header := c.GetHeader("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.Parse(h.cfg.JWTSecret, token)
			if err == nil {
				c.Set("claims", claims)
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) listMembers(c *gin.Context) {
	items, err := h.svc.ListMembers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createMember(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.CreateMember(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) updateMember(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name   string `json:"name" binding:"required"`
		NameTh string `json:"name_th"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.svc.UpdateMember(c.Request.Context(), id, req.Name, req.NameTh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) listAccounts(c *gin.Context) {
	items, err := h.svc.ListAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createAccount(c *gin.Context) {
	var req models.Account
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	a, err := h.svc.CreateAccount(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *Handler) listCategories(c *gin.Context) {
	items, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createCategory(c *gin.Context) {
	var req models.Category
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

func (h *Handler) listTransactions(c *gin.Context) {
	items, err := h.svc.ListTransactions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createTransaction(c *gin.Context) {
	var req struct {
		AccountID       *uuid.UUID  `json:"account_id"`
		CategoryID      *uuid.UUID  `json:"category_id"`
		PaidByMemberID  *uuid.UUID  `json:"paid_by_member_id"`
		IsJoint         bool        `json:"is_joint"`
		TransactionDate string      `json:"transaction_date" binding:"required"`
		Description     string      `json:"description"`
		AmountSatang    int64       `json:"amount_satang"`
		ReceiptPath     *string     `json:"receipt_path"`
		SplitParts      []int       `json:"split_parts"`
		SplitMemberIDs  []uuid.UUID `json:"split_member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction_date must be YYYY-MM-DD"})
		return
	}
	if req.IsJoint && req.AccountID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required for joint transactions"})
		return
	}
	t, err := h.svc.CreateTransaction(c.Request.Context(), service.CreateTransactionRequest{
		AccountID:       req.AccountID,
		CategoryID:      req.CategoryID,
		PaidByMemberID:  req.PaidByMemberID,
		IsJoint:         req.IsJoint,
		TransactionDate: date,
		Description:     req.Description,
		AmountSatang:    req.AmountSatang,
		ReceiptPath:     req.ReceiptPath,
		SplitParts:      req.SplitParts,
		SplitMemberIDs:  req.SplitMemberIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *Handler) bulkUpdateTransactions(c *gin.Context) {
	var req struct {
		Updates []struct {
			ID              uuid.UUID       `json:"id" binding:"required"`
			AccountID       *uuid.UUID      `json:"account_id"`
			CategoryID      *uuid.UUID      `json:"category_id"`
			PaidByMemberID  *uuid.UUID      `json:"paid_by_member_id"`
			IsJoint         *bool           `json:"is_joint"`
			TransactionDate *string         `json:"transaction_date"`
			Description     *string         `json:"description"`
			AmountSatang    *int64          `json:"amount_satang"`
			Splits          []models.Split  `json:"splits"`
			UpdateSplits    bool            `json:"update_splits"`
		} `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var items []service.BulkUpdateItem
	for _, u := range req.Updates {
		item := service.BulkUpdateItem{
			ID:             u.ID,
			AccountID:      u.AccountID,
			CategoryID:     u.CategoryID,
			PaidByMemberID: u.PaidByMemberID,
			IsJoint:        u.IsJoint,
			Description:    u.Description,
			AmountSatang: u.AmountSatang,
			Splits:       u.Splits,
			UpdateSplits: u.UpdateSplits,
		}
		if u.TransactionDate != nil {
			d, err := time.Parse("2006-01-02", *u.TransactionDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "transaction_date must be YYYY-MM-DD"})
				return
			}
			item.TransactionDate = &d
		}
		items = append(items, item)
	}

	updated, err := h.svc.BulkUpdateTransactions(c.Request.Context(), items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) listBalanceSnapshots(c *gin.Context) {
	accountID, err := uuid.Parse(c.Query("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	items, err := h.svc.ListBalanceSnapshots(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createBalanceSnapshot(c *gin.Context) {
	var req struct {
		AccountID     uuid.UUID `json:"account_id" binding:"required"`
		SnapshotDate  string    `json:"snapshot_date" binding:"required"`
		BalanceSatang int64     `json:"balance_satang" binding:"required"`
		Note          *string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", req.SnapshotDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_date must be YYYY-MM-DD"})
		return
	}
	b, err := h.svc.CreateBalanceSnapshot(c.Request.Context(), models.BalanceSnapshot{
		AccountID:     req.AccountID,
		SnapshotDate:  date,
		BalanceSatang: req.BalanceSatang,
		Note:          req.Note,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (h *Handler) listCCBalances(c *gin.Context) {
	accountID, err := uuid.Parse(c.Query("account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	items, err := h.svc.ListCCBalances(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createCCBalance(c *gin.Context) {
	var req struct {
		AccountID     uuid.UUID `json:"account_id" binding:"required"`
		StatementDate string    `json:"statement_date" binding:"required"`
		BalanceSatang int64     `json:"balance_satang" binding:"required"`
		Note          *string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", req.StatementDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "statement_date must be YYYY-MM-DD"})
		return
	}
	b, err := h.svc.CreateCCBalance(c.Request.Context(), models.CCStatementBalance{
		AccountID:     req.AccountID,
		StatementDate: date,
		BalanceSatang: req.BalanceSatang,
		Note:          req.Note,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (h *Handler) listSettlements(c *gin.Context) {
	items, err := h.svc.ListSettlements(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createSettlement(c *gin.Context) {
	var req struct {
		FromMemberID   uuid.UUID `json:"from_member_id" binding:"required"`
		ToMemberID     uuid.UUID `json:"to_member_id" binding:"required"`
		SettlementDate string    `json:"settlement_date" binding:"required"`
		AmountSatang   int64     `json:"amount_satang" binding:"required"`
		Note           *string   `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	date, err := time.Parse("2006-01-02", req.SettlementDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "settlement_date must be YYYY-MM-DD"})
		return
	}
	st, err := h.svc.CreateSettlement(c.Request.Context(), models.Settlement{
		FromMemberID:   req.FromMemberID,
		ToMemberID:     req.ToMemberID,
		SettlementDate: date,
		AmountSatang:   req.AmountSatang,
		Note:           req.Note,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, st)
}

func (h *Handler) settlementReport(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	report, err := h.svc.SettlementReport(c.Request.Context(), month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) effectiveBalance(c *gin.Context) {
	checkingID, err := uuid.Parse(c.Query("checking_account_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "checking_account_id required"})
		return
	}
	var ccID *uuid.UUID
	if s := c.Query("cc_account_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cc_account_id"})
			return
		}
		ccID = &id
	}
	report, err := h.svc.EffectiveBalance(c.Request.Context(), checkingID, ccID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) listSavingsGoals(c *gin.Context) {
	items, err := h.svc.ListSavingsGoals(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createSavingsGoal(c *gin.Context) {
	var req struct {
		Name          string     `json:"name" binding:"required"`
		TargetSatang  int64      `json:"target_satang" binding:"required"`
		CurrentSatang int64      `json:"current_satang"`
		Deadline      *string    `json:"deadline"`
		CategoryID    *uuid.UUID `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g := models.SavingsGoal{
		Name:          req.Name,
		TargetSatang:  req.TargetSatang,
		CurrentSatang: req.CurrentSatang,
		CategoryID:    req.CategoryID,
	}
	if req.Deadline != nil {
		d, err := time.Parse("2006-01-02", *req.Deadline)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "deadline must be YYYY-MM-DD"})
			return
		}
		g.Deadline = &d
	}
	goal, err := h.svc.CreateSavingsGoal(c.Request.Context(), g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, goal)
}

func (h *Handler) updateSavingsGoal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name          string     `json:"name" binding:"required"`
		TargetSatang  int64      `json:"target_satang" binding:"required"`
		CurrentSatang int64      `json:"current_satang"`
		Deadline      *string    `json:"deadline"`
		CategoryID    *uuid.UUID `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g := models.SavingsGoal{
		ID:            id,
		Name:          req.Name,
		TargetSatang:  req.TargetSatang,
		CurrentSatang: req.CurrentSatang,
		CategoryID:    req.CategoryID,
	}
	if req.Deadline != nil {
		d, err := time.Parse("2006-01-02", *req.Deadline)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "deadline must be YYYY-MM-DD"})
			return
		}
		g.Deadline = &d
	}
	goal, err := h.svc.UpdateSavingsGoal(c.Request.Context(), g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goal)
}

func (h *Handler) uploadReceipt(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	if err := os.MkdirAll(h.cfg.ReceiptsDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	name := uuid.New().String() + filepath.Ext(file.Filename)
	dest := filepath.Join(h.cfg.ReceiptsDir, name)
	if err := c.SaveUploadedFile(file, dest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"path": dest, "filename": name})
}
