package discord

import (
	"fmt"
	"net/url"
	"time"
)

type Config struct {
	WebhookURL string        `env:"DISCORD_WEBHOOK_URL"`
	Timeout    time.Duration `env:"DISCORD_WEBHOOK_TIMEOUT" envDefault:"10s"`
	Username   string        `env:"DISCORD_WEBHOOK_USERNAME" envDefault:"SARP Scrapper"`
}

func (c Config) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("DISCORD_WEBHOOK_TIMEOUT must be greater than zero")
	}
	if c.Username == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_USERNAME must not be empty")
	}
	if c.WebhookURL == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(c.WebhookURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_URL must be an absolute HTTPS URL")
	}
	return nil
}
