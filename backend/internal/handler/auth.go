package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/auth"
	"github.com/hunterwilliams/familyfinance/internal/models"
)

func (h *Handler) devLogin(c *gin.Context) {
	if !h.cfg.SkipAuth {
		c.JSON(http.StatusForbidden, gin.H{"error": "dev login disabled"})
		return
	}
	hh, err := h.svc.EnsureHousehold(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	members, err := h.svc.ListMembers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var memberID uuid.UUID
	name := "Dev User"
	if len(members) > 0 {
		memberID = members[0].ID
		name = members[0].Name
	}
	token, err := auth.Issue(h.cfg.JWTSecret, auth.Claims{
		LineUserID:  "dev-user",
		HouseholdID: hh.ID,
		MemberID:    memberID,
		DisplayName: name,
		Dev:         true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": models.AuthUser{
			LineUserID:  "dev-user",
			HouseholdID: hh.ID,
			MemberID:    &memberID,
			DisplayName: name,
		},
	})
}

func (h *Handler) lineLoginStart(c *gin.Context) {
	if h.cfg.LineLoginChannelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "LINE Login is not configured"})
		return
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", h.cfg.LineLoginChannelID)
	q.Set("redirect_uri", h.cfg.LineLoginCallbackURL)
	q.Set("state", "ff")
	q.Set("scope", "profile openid")
	c.Redirect(http.StatusFound, "https://access.line.me/oauth2/v2.1/authorize?"+q.Encode())
}

func (h *Handler) lineLoginCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", h.cfg.LineLoginCallbackURL)
	form.Set("client_id", h.cfg.LineLoginChannelID)
	form.Set("client_secret", h.cfg.LineLoginChannelSecret)
	res, err := http.PostForm("https://api.line.me/oauth2/v2.1/token", form)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"error": string(body)})
		return
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	preq, _ := http.NewRequest(http.MethodGet, "https://api.line.me/v2/profile", nil)
	preq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	pres, err := http.DefaultClient.Do(preq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer pres.Body.Close()
	pbody, _ := io.ReadAll(pres.Body)
	var profile struct {
		UserID      string `json:"userId"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(pbody, &profile); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	hh, err := h.svc.EnsureHousehold(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ident := models.LineIdentity{
		LineUserID:  profile.UserID,
		HouseholdID: hh.ID,
		DisplayName: profile.DisplayName,
	}
	if err := h.svc.Repo().UpsertLineIdentity(c.Request.Context(), ident); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	stored, _ := h.svc.Repo().GetLineIdentity(c.Request.Context(), profile.UserID)
	memberID := uuid.Nil
	if stored != nil && stored.MemberID != nil {
		memberID = *stored.MemberID
	}
	token, err := auth.Issue(h.cfg.JWTSecret, auth.Claims{
		LineUserID:  profile.UserID,
		HouseholdID: hh.ID,
		MemberID:    memberID,
		DisplayName: profile.DisplayName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	redirect := fmt.Sprintf("%s/?token=%s", h.cfg.WebBaseURL, url.QueryEscape(token))
	c.Redirect(http.StatusFound, redirect)
}

func (h *Handler) authMe(c *gin.Context) {
	claims, _ := c.Get("claims")
	if h.cfg.SkipAuth && claims == nil {
		c.JSON(http.StatusOK, models.AuthUser{DisplayName: "Dev User", LineUserID: "dev-user"})
		return
	}
	cl, ok := claims.(auth.Claims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user := models.AuthUser{
		LineUserID:  cl.LineUserID,
		HouseholdID: cl.HouseholdID,
		DisplayName: cl.DisplayName,
		NeedsLink:   cl.MemberID == uuid.Nil && !cl.Dev,
	}
	if cl.MemberID != uuid.Nil {
		id := cl.MemberID
		user.MemberID = &id
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) linkMember(c *gin.Context) {
	var req struct {
		MemberID uuid.UUID `json:"member_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims, ok := c.Get("claims")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	cl := claims.(auth.Claims)
	if err := h.svc.Repo().BindLineMember(c.Request.Context(), cl.LineUserID, req.MemberID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cl.MemberID = req.MemberID
	token, err := auth.Issue(h.cfg.JWTSecret, cl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
