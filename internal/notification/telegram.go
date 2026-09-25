package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type TelegramClient struct {
	token       string
	adminChatID string
	http        *http.Client
}

func NewTelegramClient(token, adminChatID string) *TelegramClient {
	return &TelegramClient{token: token, adminChatID: adminChatID, http: &http.Client{Timeout: 10 * time.Second}}
}

func (t *TelegramClient) Enabled() bool {
	return t.token != ""
}

func (t *TelegramClient) SendMessage(ctx context.Context, chatID any, text string) error {
	if !t.Enabled() {
		return errors.New("telegram bot token not configured")
	}
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text, "parse_mode": "HTML"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := t.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var out struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}
	json.NewDecoder(res.Body).Decode(&out)
	if !out.Ok {
		return fmt.Errorf("telegram: %s", out.Description)
	}
	return nil
}

func (t *TelegramClient) SendAdmin(ctx context.Context, text string) error {
	if t.adminChatID == "" {
		return nil
	}
	return t.SendMessage(ctx, t.adminChatID, text)
}
