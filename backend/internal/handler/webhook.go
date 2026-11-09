package handler

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hunterwilliams/familyfinance/internal/bot"
	"github.com/hunterwilliams/familyfinance/internal/linebot"
)

func (h *Handler) lineWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	secret := h.cfg.LineChannelSecret
	token := h.cfg.LineChannelAccessToken
	if secret == "" || token == "" {
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
	}
	sig := c.GetHeader("X-Line-Signature")
	if secret != "" && !linebot.VerifySignature(secret, string(body), sig) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var payload linebot.WebhookBody
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	client := linebot.NewClient(token)
	router := &bot.Router{Svc: h.svc}
	openaiKey, openaiModel := h.openaiCreds(c)

	for _, ev := range payload.Events {
		if ev.Type != "message" || ev.Message == nil {
			if ev.Type == "join" && ev.Source.GroupID != "" {
				hh, err := h.svc.EnsureHousehold(c.Request.Context())
				if err == nil {
					_ = h.svc.Repo().BindLineGroup(c.Request.Context(), ev.Source.GroupID, hh.ID)
					_ = client.Reply(ev.ReplyToken, "Bound this group to the household. Each person: send /iam 1 or /iam 2 in 1:1 chat or here.")
				}
			}
			continue
		}
		userID := ev.Source.UserID
		chatID := ev.Source.GroupID
		if chatID == "" {
			chatID = ev.Source.RoomID
		}
		if chatID == "" {
			chatID = userID
		}
		var reply string
		switch ev.Message.Type {
		case "text":
			reply, err = router.HandleText(c.Request.Context(), userID, chatID, ev.Message.Text, openaiKey, openaiModel)
		case "image":
			data, mime, gerr := client.GetContent(ev.Message.ID)
			if gerr != nil {
				reply = "Could not download that image."
			} else {
				reply, err = router.HandleImage(c.Request.Context(), userID, chatID, openaiKey, openaiModel, base64.StdEncoding.EncodeToString(data), mime)
			}
		default:
			reply = "Send a photo or a text expense."
		}
		if err != nil {
			reply = "Something went wrong."
		}
		if ev.ReplyToken != "" && reply != "" {
			_ = client.Reply(ev.ReplyToken, reply)
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
