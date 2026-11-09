package linebot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
