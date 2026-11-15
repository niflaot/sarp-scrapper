package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/server"
)

func TestLoadDefaults(t *testing.T) {
	keys := []string{"APP_ENVIRONMENT", "BIND_HOST", "BIND_PORT", "LOG_JSON", "LOG_LEVEL", "SARP_SCRAPER_ENABLED", "API_TOKEN", "SARP_API_BASE_URL", "SARP_GTA_ROL_API_BASE_URL", "SARP_API_TIMEOUT", "SARP_API_USER_AGENT", "SARP_FLARESOLVERR_URL", "SARP_FLARESOLVERR_TIMEOUT", "SARP_EMAIL", "SARP_PASSWORD", "SARP_LOGIN_URL", "SARP_MAP_URL", "SARP_BROWSER_PROFILE_DIR", "SARP_BROWSER_HEADLESS", "SARP_CHROME_PATH", "SARP_CHROME_REMOTE_DEBUG_URL", "SARP_LOGIN_TIMEOUT", "SARP_MAP_SETTLE_TIME", "SARP_SCRAPE_INTERVAL", "SARP_SCRAPE_JITTER", "SARP_EMAIL_SELECTOR", "SARP_PASSWORD_SELECTOR", "SARP_SUBMIT_SELECTOR", "SARP_2FA_SELECTOR", "SARP_2FA_SUBMIT_SELECTOR", "DISCORD_WEBHOOK_URL", "DISCORD_WEBHOOK_TIMEOUT", "DISCORD_WEBHOOK_USERNAME", "LISTINGS_STATE_FILE", "LISTINGS_HOUSE_MIN_VALUE", "LISTINGS_BUSINESS_MIN_VALUE", "LISTINGS_WAREHOUSE_MIN_VALUE", "DISCORD_BOT_TOKEN", "PRESENCE_WATCH_NAMES", "PRESENCE_DISCORD_USER_ID", "PRESENCE_POLL_INTERVAL", "PRESENCE_WATCH_ACCOUNT_IDS", "PRESENCE_ACCOUNT_POLL_INTERVAL"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	// Empty variables do not activate envDefault, so restore the values whose
	// parsers cannot accept empty input and explicitly test the composed result.
	t.Setenv("APP_ENVIRONMENT", "development")
	t.Setenv("BIND_HOST", "0.0.0.0")
	t.Setenv("BIND_PORT", "8080")
	t.Setenv("LOG_JSON", "false")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("SARP_SCRAPER_ENABLED", "false")
	t.Setenv("SARP_API_BASE_URL", "https://api.sarp.es/api")
	t.Setenv("SARP_GTA_ROL_API_BASE_URL", "https://api.gta-rol.com/api")
	t.Setenv("SARP_API_TIMEOUT", "30s")
	t.Setenv("SARP_API_USER_AGENT", "sarp-scrapper/1.0")
	t.Setenv("SARP_FLARESOLVERR_TIMEOUT", "90s")
	t.Setenv("DISCORD_WEBHOOK_TIMEOUT", "10s")
	t.Setenv("DISCORD_WEBHOOK_USERNAME", "SARP Scrapper")
	t.Setenv("LISTINGS_STATE_FILE", ".cache/listings-state.json")
	t.Setenv("LISTINGS_HOUSE_MIN_VALUE", "500000")
	t.Setenv("LISTINGS_BUSINESS_MIN_VALUE", "100000")
	t.Setenv("LISTINGS_WAREHOUSE_MIN_VALUE", "100000")
	t.Setenv("PRESENCE_POLL_INTERVAL", "60s")
	t.Setenv("PRESENCE_ACCOUNT_POLL_INTERVAL", "5m")
	t.Setenv("SARP_SCRAPE_INTERVAL", "30m")
	t.Setenv("SARP_SCRAPE_JITTER", "5m")
	t.Setenv("SARP_LOGIN_TIMEOUT", "45s")
	t.Setenv("SARP_MAP_SETTLE_TIME", "8s")

	config, err := load(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if config.Server.Environment != server.EnvironmentDevelopment || config.SARP.Interval != 30*time.Minute || config.SARP.Jitter != 5*time.Minute || config.SARP.Enabled || config.Review.HouseMinValue != 500000 || config.Review.BusinessMinValue != 100000 || config.Review.WarehouseMinValue != 100000 {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestLoadRejectsEnabledScraperWithoutCredentials(t *testing.T) {
	t.Setenv("SARP_SCRAPER_ENABLED", "true")
	t.Setenv("SARP_EMAIL", "")
	t.Setenv("SARP_PASSWORD", "")
	if _, err := load(filepath.Join(t.TempDir(), "missing.env")); err == nil {
		t.Fatal("load() expected an error")
	}
}
