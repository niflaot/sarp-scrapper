package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
)

const maxWebhookResponseSize = 1 << 20

type ListingNotice struct {
	Kind    string
	Key     string
	Name    string
	Value   int64
	IconURL string
}

type Client struct {
	config     Config
	log        *zap.Logger
	httpClient *http.Client
	apiBaseURL string

	dmMu       sync.Mutex
	dmChannels map[string]string
}

func NewClient(config Config, log *zap.Logger) *Client {
	return &Client{
		config: config, log: log, httpClient: &http.Client{Timeout: config.Timeout},
		apiBaseURL: defaultAPIBaseURL, dmChannels: make(map[string]string),
	}
}

func (c *Client) Enabled() bool { return c.config.WebhookURL != "" }

func (c *Client) NotifyListing(ctx context.Context, notice ListingNotice) error {
	if !c.Enabled() {
		return nil
	}
	payload := webhookPayload{
		Content:         fmt.Sprintf("@everyone %s disponible: **%s** — **$%s**.", newListingLabel(notice.Kind), escapeMarkdown(notice.Name), formatNumber(notice.Value)),
		Username:        c.config.Username,
		AllowedMentions: allowedMentions{Parse: []string{"everyone"}},
		Embeds: []webhookEmbed{{
			Title: notice.Name,
			Color: 0x3B82F6,
			Fields: []webhookField{
				{Name: "Tipo", Value: notice.Kind, Inline: true},
				{Name: "Valor", Value: "$" + formatNumber(notice.Value), Inline: true},
				{Name: "Llave", Value: notice.Key, Inline: true},
			},
		}},
	}
	if isHTTPSURL(notice.IconURL) {
		payload.Embeds[0].Thumbnail = &webhookThumbnail{URL: notice.IconURL}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Discord property notification: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Discord property notification: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send Discord property notification: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxWebhookResponseSize+1))
	if err != nil {
		return fmt.Errorf("read Discord property notification response: %w", err)
	}
	if len(responseBody) > maxWebhookResponseSize {
		return fmt.Errorf("Discord property notification response exceeds %d bytes", maxWebhookResponseSize)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Discord property notification returned HTTP %d", response.StatusCode)
	}
	c.log.Debug("Discord listing notification sent", zap.String("kind", notice.Kind), zap.String("listing_key", notice.Key))
	return nil
}

type webhookPayload struct {
	Content         string          `json:"content"`
	Username        string          `json:"username"`
	AllowedMentions allowedMentions `json:"allowed_mentions"`
	Embeds          []webhookEmbed  `json:"embeds"`
}

type allowedMentions struct {
	Parse []string `json:"parse"`
}

type webhookEmbed struct {
	Title     string            `json:"title"`
	Color     int               `json:"color"`
	Fields    []webhookField    `json:"fields"`
	Thumbnail *webhookThumbnail `json:"thumbnail,omitempty"`
}

type webhookField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type webhookThumbnail struct {
	URL string `json:"url"`
}

func formatNumber(value int64) string {
	digits := strconv.FormatInt(value, 10)
	start := 0
	if len(digits) > 0 && digits[0] == '-' {
		start = 1
	}
	for index := len(digits) - 3; index > start; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}

func isHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "*", "\\*", "_", "\\_", "`", "\\`", "~", "\\~", "|", "\\|")
	return replacer.Replace(value)
}

func newListingLabel(kind string) string {
	switch {
	case strings.EqualFold(kind, "Propiedad"):
		return "Nueva propiedad"
	case strings.EqualFold(kind, "Almacén"):
		return "Nuevo almacén"
	default:
		return "Nuevo negocio"
	}
}
