// Package sarp owns the isolated browser used for the authorized SARP session.
package sarp

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Enabled                 bool          `env:"SARP_SCRAPER_ENABLED" envDefault:"false"`
	APIToken                string        `env:"API_TOKEN"`
	APIBaseURL              string        `env:"SARP_API_BASE_URL" envDefault:"https://api.sarp.es/api"`
	GTARolAPIBaseURL        string        `env:"SARP_GTA_ROL_API_BASE_URL" envDefault:"https://api.gta-rol.com/api"`
	APITimeout              time.Duration `env:"SARP_API_TIMEOUT" envDefault:"30s"`
	APIUserAgent            string        `env:"SARP_API_USER_AGENT" envDefault:"sarp-scrapper/1.0"`
	FlareSolverrURL         string        `env:"SARP_FLARESOLVERR_URL"`
	FlareSolverrTimeout     time.Duration `env:"SARP_FLARESOLVERR_TIMEOUT" envDefault:"90s"`
	Email                   string        `env:"SARP_EMAIL"`
	Password                string        `env:"SARP_PASSWORD"`
	LoginURL                string        `env:"SARP_LOGIN_URL" envDefault:"https://ucp.sarp.es/"`
	MapURL                  string        `env:"SARP_MAP_URL" envDefault:"https://ucp.sarp.es/app/ucp/map"`
	ProfileDir              string        `env:"SARP_BROWSER_PROFILE_DIR" envDefault:".sarp-browser"`
	ChromePath              string        `env:"SARP_CHROME_PATH"`
	ChromeRemoteDebugURL    string        `env:"SARP_CHROME_REMOTE_DEBUG_URL"`
	Headless                bool          `env:"SARP_BROWSER_HEADLESS" envDefault:"false"`
	LoginTimeout            time.Duration `env:"SARP_LOGIN_TIMEOUT" envDefault:"5m"`
	MapSettleTime           time.Duration `env:"SARP_MAP_SETTLE_TIME" envDefault:"8s"`
	Interval                time.Duration `env:"SARP_SCRAPE_INTERVAL" envDefault:"30m"`
	Jitter                  time.Duration `env:"SARP_SCRAPE_JITTER" envDefault:"5m"`
	EmailSelector           string        `env:"SARP_EMAIL_SELECTOR" envDefault:"input[type=email],input[name=email],input[name=username],input[name=login],input[autocomplete=username],input[placeholder*='correo' i],input[placeholder*='email' i]"`
	PasswordSelector        string        `env:"SARP_PASSWORD_SELECTOR" envDefault:"input[type=password],input[name=password]"`
	SubmitSelector          string        `env:"SARP_SUBMIT_SELECTOR" envDefault:"button[type=submit],input[type=submit]"`
	TwoFactorSelector       string        `env:"SARP_2FA_SELECTOR" envDefault:"input[autocomplete=one-time-code],input[name*=otp],input[name*=code],input[inputmode=numeric]"`
	TwoFactorSubmitSelector string        `env:"SARP_2FA_SUBMIT_SELECTOR" envDefault:"button[type=submit],input[type=submit]"`
}

func (c Config) Validate() error {
	if c.Interval <= 0 {
		return fmt.Errorf("SARP_SCRAPE_INTERVAL must be greater than zero")
	}
	if c.Jitter < 0 || c.Jitter >= c.Interval {
		return fmt.Errorf("SARP_SCRAPE_JITTER must be non-negative and less than SARP_SCRAPE_INTERVAL")
	}
	if c.LoginTimeout <= 0 {
		return fmt.Errorf("SARP_LOGIN_TIMEOUT must be greater than zero")
	}
	if c.MapSettleTime < 0 {
		return fmt.Errorf("SARP_MAP_SETTLE_TIME must be non-negative")
	}
	if c.APITimeout <= 0 {
		return fmt.Errorf("SARP_API_TIMEOUT must be greater than zero")
	}
	if err := validateHTTPURL("SARP_API_BASE_URL", c.APIBaseURL); err != nil {
		return err
	}
	if err := validateHTTPURL("SARP_GTA_ROL_API_BASE_URL", c.GTARolAPIBaseURL); err != nil {
		return err
	}
	if c.APIUserAgent == "" {
		return fmt.Errorf("SARP_API_USER_AGENT must not be empty")
	}
	if c.FlareSolverrTimeout <= 0 {
		return fmt.Errorf("SARP_FLARESOLVERR_TIMEOUT must be greater than zero")
	}
	if c.FlareSolverrURL != "" {
		if err := validateHTTPURL("SARP_FLARESOLVERR_URL", c.FlareSolverrURL); err != nil {
			return err
		}
	}
	if c.ChromeRemoteDebugURL != "" {
		if err := validateLoopbackHTTPURL("SARP_CHROME_REMOTE_DEBUG_URL", c.ChromeRemoteDebugURL); err != nil {
			return err
		}
		if err := validateHTTPURL("SARP_MAP_URL", c.MapURL); err != nil {
			return err
		}
	}
	if !c.Enabled {
		return nil
	}
	if c.APIToken != "" {
		return nil
	}
	if c.Email == "" {
		return fmt.Errorf("SARP_EMAIL must not be empty when scraping is enabled")
	}
	if c.Password == "" {
		return fmt.Errorf("SARP_PASSWORD must not be empty when scraping is enabled")
	}
	if c.ProfileDir == "" {
		return fmt.Errorf("SARP_BROWSER_PROFILE_DIR must not be empty")
	}
	for name, value := range map[string]string{"SARP_LOGIN_URL": c.LoginURL, "SARP_MAP_URL": c.MapURL} {
		if err := validateHTTPURL(name, value); err != nil {
			return err
		}
	}
	selectors := map[string]string{
		"SARP_EMAIL_SELECTOR": c.EmailSelector, "SARP_PASSWORD_SELECTOR": c.PasswordSelector,
		"SARP_SUBMIT_SELECTOR": c.SubmitSelector, "SARP_2FA_SELECTOR": c.TwoFactorSelector,
		"SARP_2FA_SUBMIT_SELECTOR": c.TwoFactorSubmitSelector,
	}
	for name, selector := range selectors {
		if selector == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}

func validateLoopbackHTTPURL(name, value string) error {
	if err := validateHTTPURL(name, value); err != nil {
		return err
	}
	parsed, _ := url.Parse(value)
	hostname := strings.ToLower(parsed.Hostname())
	if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
		return fmt.Errorf("%s must point to a loopback address", name)
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	return nil
}
