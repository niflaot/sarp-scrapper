package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

const defaultAPIBaseURL = "https://discord.com/api/v10"

// PresenceNotice describes a watched character that just connected.
type PresenceNotice struct {
	RecipientID string
	CharacterID int64
	Character   string
	Level       int
	Account     string
}

type mentions struct {
	Parse []string `json:"parse"`
	Users []string `json:"users,omitempty"`
}

type messagePayload struct {
	Content         string   `json:"content"`
	Username        string   `json:"username,omitempty"`
	AllowedMentions mentions `json:"allowed_mentions"`
}

// PresenceEnabled reports whether a connection notice has any delivery channel:
// a direct message through the bot, or a mention through the webhook.
func (c *Client) PresenceEnabled() bool { return c.config.BotToken != "" || c.Enabled() }

// AccountNotice describes a watched SARP account that appeared as online.
type AccountNotice struct {
	RecipientID string
	AccountID   int64
	AccountName string
	AdminLevel  int
}

// NotifyPresence tells the recipient that a watched character connected.
func (c *Client) NotifyPresence(ctx context.Context, notice PresenceNotice) error {
	text := fmt.Sprintf("🟢 **%s** se acaba de conectar (nivel %d, cuenta %s, ID %d).",
		escapeMarkdown(notice.Character), notice.Level, escapeMarkdown(notice.Account), notice.CharacterID)
	return c.deliver(ctx, notice.RecipientID, text)
}

// NotifyAccountPresence tells the recipient that a watched account is online.
func (c *Client) NotifyAccountPresence(ctx context.Context, notice AccountNotice) error {
	text := fmt.Sprintf("🟢 La cuenta **%s** (ID %d) aparece conectada.", escapeMarkdown(notice.AccountName), notice.AccountID)
	if notice.AdminLevel > 0 {
		text = fmt.Sprintf("🟢 La cuenta **%s** (ID %d, admin nivel %d) aparece conectada.",
			escapeMarkdown(notice.AccountName), notice.AccountID, notice.AdminLevel)
	}
	return c.deliver(ctx, notice.RecipientID, text)
}

// deliver sends a direct message with the bot (when DISCORD_BOT_TOKEN is set)
// and a mention through the webhook (when DISCORD_WEBHOOK_URL is set). It
// succeeds when at least one channel delivered the text, so a closed DM does
// not cause the webhook mention to be repeated on every retry.
func (c *Client) deliver(ctx context.Context, recipientID, text string) error {
	var failures []error
	delivered := false
	if c.config.BotToken != "" {
		if err := c.sendDirectMessage(ctx, recipientID, text); err != nil {
			failures = append(failures, err)
		} else {
			delivered = true
		}
	}
	if c.Enabled() {
		payload := messagePayload{
			Content:         fmt.Sprintf("<@%s> %s", recipientID, text),
			Username:        c.config.Username,
			AllowedMentions: mentions{Parse: []string{}, Users: []string{recipientID}},
		}
		if err := c.postJSON(ctx, c.config.WebhookURL, "", payload, nil, "Discord presence webhook"); err != nil {
			failures = append(failures, err)
		} else {
			delivered = true
		}
	}
	if delivered {
		for _, err := range failures {
			c.log.Warn("Discord presence notification channel failed", zap.Error(err))
		}
		return nil
	}
	return errors.Join(failures...)
}

func (c *Client) sendDirectMessage(ctx context.Context, userID, content string) error {
	channelID, err := c.directChannel(ctx, userID)
	if err != nil {
		return err
	}
	payload := messagePayload{Content: content, AllowedMentions: mentions{Parse: []string{}}}
	endpoint := c.apiBaseURL + "/channels/" + url.PathEscape(channelID) + "/messages"
	return c.postJSON(ctx, endpoint, "Bot "+c.config.BotToken, payload, nil, "Discord direct message")
}

func (c *Client) directChannel(ctx context.Context, userID string) (string, error) {
	c.dmMu.Lock()
	defer c.dmMu.Unlock()
	if id, ok := c.dmChannels[userID]; ok {
		return id, nil
	}
	var channel struct {
		ID string `json:"id"`
	}
	err := c.postJSON(ctx, c.apiBaseURL+"/users/@me/channels", "Bot "+c.config.BotToken,
		map[string]string{"recipient_id": userID}, &channel, "Discord direct channel")
	if err != nil {
		return "", err
	}
	if channel.ID == "" {
		return "", fmt.Errorf("Discord direct channel response omitted the channel id")
	}
	c.dmChannels[userID] = channel.ID
	return channel.ID, nil
}

// postJSON sends one JSON POST. Errors only carry the HTTP status, never the
// request, so the bot token and the webhook URL cannot leak into logs.
func (c *Client) postJSON(ctx context.Context, endpoint, authorization string, payload, out any, label string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s: %w", label, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create %s request: %w", label, err)
	}
	request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send %s: %w", label, urlErrorWithoutURL(err))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxWebhookResponseSize+1))
	if err != nil {
		return fmt.Errorf("read %s response: %w", label, err)
	}
	if len(data) > maxWebhookResponseSize {
		return fmt.Errorf("%s response exceeds %d bytes", label, maxWebhookResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s returned HTTP %d", label, response.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s response: %w", label, err)
		}
	}
	return nil
}

// urlErrorWithoutURL drops the request URL (which contains the webhook token)
// from transport errors.
func urlErrorWithoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
