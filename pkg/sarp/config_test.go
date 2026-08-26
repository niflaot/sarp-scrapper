package sarp

import (
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Enabled: true, Email: "user@example.com", Password: "password",
		APIBaseURL: "https://api.sarp.es/api", GTARolAPIBaseURL: "https://api.gta-rol.com/api", APITimeout: time.Second, APIUserAgent: "test-client/1.0",
		FlareSolverrTimeout: time.Second,
		LoginURL:            "https://ucp.sarp.es/", MapURL: "https://ucp.sarp.es/app/ucp/map",
		ProfileDir: ".test-browser", LoginTimeout: time.Second, MapSettleTime: time.Second,
		Interval: 30 * time.Minute, Jitter: 5 * time.Minute,
		EmailSelector: "input[type=email]", PasswordSelector: "input[type=password]",
		SubmitSelector: "button[type=submit]", TwoFactorSelector: "input[name=code]",
		TwoFactorSubmitSelector: "button[type=submit]",
	}
}

func TestConfigValidate(t *testing.T) {
	valid := validConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []func(*Config){
		func(c *Config) { c.Email = "" }, func(c *Config) { c.Password = "" },
		func(c *Config) { c.LoginURL = "relative" }, func(c *Config) { c.MapURL = "" },
		func(c *Config) { c.ProfileDir = "" }, func(c *Config) { c.Interval = 0 },
		func(c *Config) { c.Jitter = c.Interval }, func(c *Config) { c.LoginTimeout = 0 },
		func(c *Config) { c.MapSettleTime = -1 }, func(c *Config) { c.TwoFactorSelector = "" },
		func(c *Config) { c.APITimeout = 0 }, func(c *Config) { c.APIBaseURL = "relative" },
		func(c *Config) { c.GTARolAPIBaseURL = "relative" },
		func(c *Config) { c.APIUserAgent = "" },
		func(c *Config) { c.FlareSolverrTimeout = 0 }, func(c *Config) { c.FlareSolverrURL = "relative" },
	}
	for index, mutate := range tests {
		config := validConfig()
		mutate(&config)
		if err := config.Validate(); err == nil {
			t.Fatalf("case %d expected error: %#v", index, config)
		}
	}

	disabled := validConfig()
	disabled.Enabled, disabled.Email, disabled.Password = false, "", ""
	if err := disabled.Validate(); err != nil {
		t.Fatalf("disabled Validate() error = %v", err)
	}

	apiOnly := validConfig()
	apiOnly.APIToken, apiOnly.Email, apiOnly.Password = "sarp_test", "", ""
	if err := apiOnly.Validate(); err != nil {
		t.Fatalf("API-only Validate() error = %v", err)
	}

	apiOnly.ChromeRemoteDebugURL = "http://127.0.0.1:9222"
	if err := apiOnly.Validate(); err != nil {
		t.Fatalf("loopback Chrome URL Validate() error = %v", err)
	}
	apiOnly.ChromeRemoteDebugURL = "http://example.com:9222"
	if err := apiOnly.Validate(); err == nil {
		t.Fatal("external Chrome URL Validate() expected an error")
	}
}
