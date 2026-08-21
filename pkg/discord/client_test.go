package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestClientNotifyProperty(t *testing.T) {
	var received webhookPayload
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{WebhookURL: server.URL, Timeout: time.Second, Username: "test"}, zap.NewNop())
	client.httpClient = server.Client()
	if err := client.NotifyListing(context.Background(), ListingNotice{
		Kind: "Propiedad", Key: "749", Name: "Olympia Avenue", Value: 1338876, IconURL: "https://example.com/icon.png",
	}); err != nil {
		t.Fatalf("NotifyListing() error = %v", err)
	}
	if received.Content != "@everyone Nueva propiedad disponible: **Olympia Avenue** — **$1,338,876**." || len(received.AllowedMentions.Parse) != 1 || received.AllowedMentions.Parse[0] != "everyone" {
		t.Fatalf("unexpected mention payload: %#v", received)
	}
	if len(received.Embeds) != 1 || received.Embeds[0].Title != "Olympia Avenue" || received.Embeds[0].Fields[1].Value != "$1,338,876" {
		t.Fatalf("unexpected embed: %#v", received.Embeds)
	}
}

func TestClientDisabled(t *testing.T) {
	client := NewClient(Config{Timeout: time.Second, Username: "test"}, zap.NewNop())
	if err := client.NotifyListing(context.Background(), ListingNotice{}); err != nil {
		t.Fatalf("NotifyListing() error = %v", err)
	}
}

func TestNewListingLabelWarehouse(t *testing.T) {
	if label := newListingLabel("Almacén"); label != "Nuevo almacén" {
		t.Fatalf("newListingLabel() = %q", label)
	}
}
