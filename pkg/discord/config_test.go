package discord

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{WebhookURL: "https://discord.com/api/webhooks/1/token", Timeout: time.Second, Username: "SARP"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	valid.WebhookURL = "http://discord.com/api/webhooks/1/token"
	if err := valid.Validate(); err == nil {
		t.Fatal("Validate() expected HTTPS error")
	}
}
