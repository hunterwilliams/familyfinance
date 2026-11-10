package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	APIKey  string
	Model   string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, model string) *Client {
	return NewWithBase(apiKey, model, "")
}

func NewWithBase(apiKey, model, baseURL string) *Client {
	apiKey = NormalizeKey(apiKey)
	if baseURL == "" {
		baseURL = BaseURLForKey(apiKey, "")
	}
	return &Client{
		APIKey:  apiKey,
		Model:   DefaultModel(apiKey, model),
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func NormalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, `"'`)
	key = strings.TrimSpace(key)
	if len(key) >= 7 && strings.EqualFold(key[:7], "bearer ") {
		key = strings.TrimSpace(key[7:])
	}
	return key
}

func isOpenRouterKey(key string) bool {
	return strings.HasPrefix(NormalizeKey(key), "sk-or-")
}

func isXAIKey(key string) bool {
	return strings.HasPrefix(NormalizeKey(key), "xai-")
}

func BaseURLForKey(key, override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimRight(strings.TrimSpace(override), "/")
	}
	switch {
	case isOpenRouterKey(key):
		return "https://openrouter.ai/api/v1"
	case isXAIKey(key):
		return "https://api.x.ai/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func DefaultModel(key, model string) string {
	model = strings.TrimSpace(model)
	switch {
	case isOpenRouterKey(key):
		if model == "" || model == "gpt-4o" {
			return "openai/gpt-4o"
		}
		if model == "gpt-4o-mini" {
			return "openai/gpt-4o-mini"
		}
		return model
	case isXAIKey(key):
		if model == "" || model == "gpt-4o" {
			return "grok-2-vision-1212"
		}
		return model
	default:
		if model == "" {
			return "gpt-4o"
		}
		return model
	}
}

type Intent struct {
	Kind        string  `json:"kind"`
	AmountTHB   float64 `json:"amount_thb"`
	Description string  `json:"description"`
	IsJoint     bool    `json:"is_joint"`
	Date        string  `json:"date"`
	Caregiver   string  `json:"caregiver"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Hours       float64 `json:"hours"`
}

type ReceiptFields struct {
	AmountTHB     float64  `json:"amount_thb"`
	Date          string   `json:"date"`
	Merchant      string   `json:"merchant"`
	AccountHolder string   `json:"account_holder"`
	NamesVisible  []string `json:"names_visible"`
}

func (c *Client) ParseIntent(ctx context.Context, text string, extraContext string) (Intent, error) {
	prompt := `You extract household finance intents from English messages.
Return JSON only with fields:
kind: one of add_expense, nanny_hours, nanny_holiday, nanny_worked_holiday, confirm, cancel, help, other
amount_thb: number
description: string
is_joint: boolean (true if paid from joint account / no reimbursement)
date: YYYY-MM-DD if mentioned (tomorrow/tmrw = next calendar day in Asia/Bangkok) else empty
caregiver: nanny name if nanny hours; use the known name when the message says nanny
start_time, end_time: HH:MM 24h
hours: number if mentioned

For nanny_hours extra pay only (not the usual shift):
- come early at a time: start_time = arrival, leave end_time empty
- stay late until a time: end_time = leave time, leave start_time empty
- an explicit extra range: set both start_time and end_time

For nanny_holiday: they take a usual work day off (personal day off). Set caregiver and date only. Paid Thai government holidays do not create hours owed.
For nanny_worked_holiday: they work on a paid government holiday and should be paid the usual shift as extra. Set caregiver and date.

`
	if strings.TrimSpace(extraContext) != "" {
		prompt += extraContext + "\n\n"
	}
	prompt += "Message: " + text

	raw, err := c.chat(ctx, []map[string]any{
		{"role": "system", "content": "Return JSON only."},
		{"role": "user", "content": prompt},
	})
	if err != nil {
		return Intent{}, err
	}
	var in Intent
	if err := json.Unmarshal([]byte(extractJSON(raw)), &in); err != nil {
		return Intent{Kind: "other", Description: text}, nil
	}
	return in, nil
}

func (c *Client) ParseReceiptImage(ctx context.Context, imageBase64, mime string, memberHint string) (ReceiptFields, error) {
	if mime == "" {
		mime = "image/jpeg"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, imageBase64)
	hint := "Household members (match account holder to these if the printed name is clearly one of them; still return the printed text): " + memberHint
	if strings.TrimSpace(memberHint) == "" {
		hint = "Capture the account or cardholder name if shown."
	}
	raw, err := c.chat(ctx, []map[string]any{
		{"role": "system", "content": "Extract receipt or bank slip data. Return JSON only: amount_thb (number), date (YYYY-MM-DD or empty), merchant (string), account_holder (account/cardholder name as printed in English or Thai; empty if none; never use the merchant), names_visible (array of person names printed on the slip)."},
		{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": "Read this receipt or payment screenshot. " + hint},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": dataURL}},
		}},
	})
	if err != nil {
		return ReceiptFields{}, err
	}
	var f ReceiptFields
	if err := json.Unmarshal([]byte(extractJSON(raw)), &f); err != nil {
		return ReceiptFields{}, fmt.Errorf("could not understand that receipt photo. Try a clearer JPEG or retake the picture")
	}
	return f, nil
}

func (c *Client) Validate(ctx context.Context) error {
	_, err := c.ListModels(ctx)
	return err
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("API key is not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach AI API: %w", err)
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, apiError(res.StatusCode, payload)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("could not parse models list")
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

func (c *Client) CheckModel(ctx context.Context) error {
	ids, err := c.ListModels(ctx)
	if err != nil {
		return err
	}
	if modelListed(ids, c.Model) {
		return nil
	}
	if len(ids) == 0 {
		return fmt.Errorf("API key works, but the provider returned no models")
	}
	return fmt.Errorf("model %s is not available with this key", c.Model)
}

func modelListed(ids []string, model string) bool {
	want := strings.ToLower(strings.TrimSpace(model))
	if want == "" {
		return false
	}
	for _, id := range ids {
		got := strings.ToLower(id)
		if got == want || strings.HasSuffix(got, "/"+want) || strings.HasSuffix(want, "/"+got) {
			return true
		}
	}
	return false
}

func (c *Client) chat(ctx context.Context, messages []map[string]any) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("API key is not set")
	}
	body := map[string]any{
		"model":       c.Model,
		"messages":    messages,
		"temperature": 0,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", apiError(res.StatusCode, payload)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func apiError(status int, payload []byte) error {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Code    any    `json:"code"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	_ = json.Unmarshal(payload, &parsed)
	msg := strings.TrimSpace(parsed.Error.Message)
	code := strings.ToLower(fmt.Sprint(parsed.Error.Code))
	lower := strings.ToLower(msg)
	if status == 401 || strings.Contains(code, "invalid_api") || strings.Contains(lower, "incorrect api key") || strings.Contains(lower, "invalid api key") {
		return fmt.Errorf("invalid API key — use OpenRouter (sk-or-...), OpenAI (sk-...), or xAI (xai-...)")
	}
	if msg != "" {
		return fmt.Errorf("openai: %s", msg)
	}
	trim := strings.TrimSpace(string(payload))
	if trim == "" {
		return fmt.Errorf("openai: HTTP %d", status)
	}
	return fmt.Errorf("openai: %s", trim)
}

func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if strings.Contains(c.BaseURL, "openrouter.ai") {
		req.Header.Set("HTTP-Referer", "https://github.com/hunterwilliams/familyfinance")
		req.Header.Set("X-Title", "Family Finance")
	}
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}
