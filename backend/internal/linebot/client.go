package linebot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	AccessToken string
	HTTP        *http.Client
}

func NewClient(token string) *Client {
	return &Client{AccessToken: token, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

func VerifySignature(channelSecret, body, signature string) bool {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write([]byte(body))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

type ReplyMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (c *Client) Reply(replyToken, text string) error {
	if c.AccessToken == "" {
		return fmt.Errorf("line access token not set")
	}
	payload := map[string]any{
		"replyToken": replyToken,
		"messages":   []ReplyMessage{{Type: "text", Text: text}},
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.line.me/v2/bot/message/reply", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("line reply: %s", body)
	}
	return nil
}

type BotInfo struct {
	UserID      string `json:"userId"`
	BasicID     string `json:"basicId"`
	PremiumID   string `json:"premiumId"`
	DisplayName string `json:"displayName"`
	ChatMode    string `json:"chatMode"`
}

func (c *Client) GetBotInfo(ctx context.Context) (BotInfo, error) {
	var info BotInfo
	if c.AccessToken == "" {
		return info, fmt.Errorf("LINE access token is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.line.me/v2/bot/info", nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return info, fmt.Errorf("could not reach LINE API: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return info, lineAPIError(res.StatusCode, body)
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return info, fmt.Errorf("could not parse LINE bot info")
	}
	return info, nil
}

func (c *Client) CountFollowers(ctx context.Context) (int, error) {
	if c.AccessToken == "" {
		return 0, fmt.Errorf("LINE access token is not set")
	}
	count := 0
	start := ""
	for i := 0; i < 10; i++ {
		u := "https://api.line.me/v2/bot/followers/ids?limit=1000"
		if start != "" {
			u += "&start=" + url.QueryEscape(start)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return count, err
		}
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return count, fmt.Errorf("could not reach LINE API: %w", err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode >= 300 {
			return count, lineAPIError(res.StatusCode, body)
		}
		var page struct {
			UserIDs []string `json:"userIds"`
			Next    string   `json:"next"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return count, fmt.Errorf("could not parse LINE friends list")
		}
		count += len(page.UserIDs)
		if page.Next == "" {
			return count, nil
		}
		start = page.Next
	}
	return count, nil
}

func lineAPIError(status int, payload []byte) error {
	var parsed struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(payload, &parsed)
	msg := parsed.Message
	if msg == "" {
		msg = string(payload)
	}
	if status == 401 {
		return fmt.Errorf("invalid LINE access token")
	}
	if msg != "" {
		return fmt.Errorf("LINE API: %s", msg)
	}
	return fmt.Errorf("LINE API: HTTP %d", status)
}

func (c *Client) GetContent(messageID string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api-data.line.me/v2/bot/message/"+messageID+"/content", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	if res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("line content: %s", b)
	}
	return b, res.Header.Get("Content-Type"), nil
}

type Event struct {
	Type       string `json:"type"`
	ReplyToken string `json:"replyToken"`
	Source     struct {
		Type    string `json:"type"`
		UserID  string `json:"userId"`
		GroupID string `json:"groupId"`
		RoomID  string `json:"roomId"`
	} `json:"source"`
	Message *struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"message"`
}

type WebhookBody struct {
	Events []Event `json:"events"`
}
