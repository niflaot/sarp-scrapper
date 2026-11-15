package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

type recorded struct{ method, path, auth, body string }

func recordingServer(t *testing.T, status map[string]int) (*httptest.Server, func() []recorded) {
	t.Helper()
	var mu sync.Mutex
	var requests []recorded
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body json.RawMessage
		_ = json.NewDecoder(request.Body).Decode(&body)
		mu.Lock()
		requests = append(requests, recorded{request.Method, request.URL.Path, request.Header.Get("Authorization"), string(body)})
		mu.Unlock()
		if code, ok := status[request.URL.Path]; ok {
			response.WriteHeader(code)
			return
		}
		if request.URL.Path == "/users/@me/channels" {
			_, _ = response.Write([]byte(`{"id":"555"}`))
			return
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	return server, func() []recorded { mu.Lock(); defer mu.Unlock(); return append([]recorded(nil), requests...) }
}

func presenceClient(server *httptest.Server, config Config) *Client {
	config.Timeout, config.Username = time.Second, "test"
	client := NewClient(config, zap.NewNop())
	client.httpClient, client.apiBaseURL = server.Client(), server.URL
	return client
}

var testNotice = PresenceNotice{RecipientID: "278642935426449418", CharacterID: 9, Character: "Harper_Brooks", Level: 12, Account: "Acc"}

func TestNotifyPresenceDirectMessageAndWebhook(t *testing.T) {
	server, requests := recordingServer(t, nil)
	client := presenceClient(server, Config{BotToken: "secret", WebhookURL: server.URL + "/webhook"})
	for attempt := 0; attempt < 2; attempt++ {
		if err := client.NotifyPresence(t.Context(), testNotice); err != nil {
			t.Fatalf("NotifyPresence() error = %v", err)
		}
	}
	got := requests()
	// Attempt 1: channel, message, webhook. Attempt 2 reuses the cached DM channel.
	if len(got) != 5 {
		t.Fatalf("requests = %#v", got)
	}
	if got[0].path != "/users/@me/channels" || got[0].auth != "Bot secret" || !strings.Contains(got[0].body, `"recipient_id":"278642935426449418"`) {
		t.Fatalf("unexpected channel request: %#v", got[0])
	}
	if got[1].path != "/channels/555/messages" || !strings.Contains(got[1].body, `Harper\\_Brooks`) || !strings.Contains(got[1].body, `"parse":[]`) {
		t.Fatalf("unexpected DM request: %#v", got[1])
	}
	var webhook messagePayload
	if err := json.Unmarshal([]byte(got[2].body), &webhook); err != nil {
		t.Fatalf("decode webhook body: %v", err)
	}
	if got[2].path != "/webhook" || got[2].auth != "" || !strings.HasPrefix(webhook.Content, "<@278642935426449418> ") ||
		len(webhook.AllowedMentions.Users) != 1 || webhook.AllowedMentions.Users[0] != "278642935426449418" || len(webhook.AllowedMentions.Parse) != 0 {
		t.Fatalf("unexpected webhook request: %#v", got[2])
	}
}

func TestNotifyPresenceSucceedsWhenOneChannelDelivers(t *testing.T) {
	server, _ := recordingServer(t, map[string]int{"/users/@me/channels": http.StatusForbidden})
	client := presenceClient(server, Config{BotToken: "secret", WebhookURL: server.URL + "/webhook"})
	if err := client.NotifyPresence(t.Context(), testNotice); err != nil {
		t.Fatalf("webhook delivered, NotifyPresence() error = %v", err)
	}

	client = presenceClient(server, Config{BotToken: "secret"})
	err := client.NotifyPresence(t.Context(), testNotice)
	if err == nil || strings.Contains(err.Error(), "secret") || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected a token-free HTTP 403 error, got %v", err)
	}
}

func TestPresenceEnabled(t *testing.T) {
	if NewClient(Config{Timeout: time.Second}, zap.NewNop()).PresenceEnabled() {
		t.Fatal("PresenceEnabled() must be false without bot token or webhook")
	}
	if !NewClient(Config{Timeout: time.Second, BotToken: "x"}, zap.NewNop()).PresenceEnabled() {
		t.Fatal("PresenceEnabled() must be true with a bot token")
	}
}

func TestNotifyAccountPresence(t *testing.T) {
	server, requests := recordingServer(t, nil)
	client := presenceClient(server, Config{BotToken: "secret"})
	if err := client.NotifyAccountPresence(t.Context(), AccountNotice{RecipientID: "278642935426449418", AccountID: 13297, AccountName: "var_ky", AdminLevel: 3}); err != nil {
		t.Fatalf("NotifyAccountPresence() error = %v", err)
	}
	got := requests()
	if len(got) != 2 || !strings.Contains(got[1].body, `var\\_ky`) || !strings.Contains(got[1].body, "13297") || !strings.Contains(got[1].body, "admin nivel 3") {
		t.Fatalf("unexpected requests: %#v", got)
	}
}
