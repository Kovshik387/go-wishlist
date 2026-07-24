package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
}

type APIError struct {
	StatusCode int
	ErrorCode  int
	Description string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram API error %d: %s", e.ErrorCode, e.Description)
}

func IsBlocked(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode == http.StatusForbidden
}

func RetryAfter(err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

type InlineKeyboardButton struct {
	Text   string `json:"text"`
	WebApp *WebAppInfo `json:"web_app,omitempty"`
	URL    string `json:"url,omitempty"`
}

type WebAppInfo struct {
	URL string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
	ChatID      int64                `json:"chat_id"`
	Text        string               `json:"text"`
	ParseMode   string               `json:"parse_mode,omitempty"`
	ReplyMarkup InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (c *Client) SendMessage(ctx context.Context, request SendMessageRequest) error {
	if c.Token == "" {
		return errors.New("Telegram bot token is not configured")
	}
	return c.call(ctx, "sendMessage", request)
}

func (c *Client) SetWebhook(ctx context.Context, webhookURL, secret string) error {
	return c.call(ctx, "setWebhook", map[string]any{
		"url": webhookURL, "secret_token": secret,
		"allowed_updates": []string{"message", "my_chat_member"},
		"drop_pending_updates": false,
	})
}

func (c *Client) call(ctx context.Context, method string, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/bot"+c.Token+"/"+method, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var envelope struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode Telegram response (status %s): %w", strconv.Itoa(resp.StatusCode), err)
	}
	if !envelope.OK {
		return &APIError{
			StatusCode: resp.StatusCode, ErrorCode: envelope.ErrorCode,
			Description: envelope.Description,
			RetryAfter: time.Duration(envelope.Parameters.RetryAfter) * time.Second,
		}
	}
	return nil
}
