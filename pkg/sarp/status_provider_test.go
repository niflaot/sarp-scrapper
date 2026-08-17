package sarp

import (
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

func TestStatusProviderUsesAvailableSource(t *testing.T) {
	api := NewAPIClient(Config{}, zap.NewNop())
	browser := NewBrowser(Config{}, zap.NewNop())
	provider := NewStatusProvider(Config{APIToken: "sarp_test"}, api, browser)

	if provider.Status() != session.StatePending {
		t.Fatalf("Status() = %q", provider.Status())
	}
	browser.setState(session.StateAvailable)
	if provider.Status() != session.StateAvailable {
		t.Fatalf("browser Status() = %q", provider.Status())
	}
	browser.setState(session.StatePending)
	api.setState(session.StateAvailable)
	if provider.Status() != session.StateAvailable {
		t.Fatalf("API Status() = %q", provider.Status())
	}
}
