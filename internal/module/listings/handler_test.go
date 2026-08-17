package listings

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/zap"
)

func TestPanelAndHTTPReviewActions(t *testing.T) {
	store, err := review.NewStore(review.Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	service := NewService(store, &notifierStub{}, zap.NewNop())
	if _, err := service.Process(t.Context(), KindBusiness, []review.Item{{Key: "9", Name: "Cluckin Bell", Value: 250000}}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(service))

	response, err := httpServer.App().Test(httptest.NewRequest(http.MethodGet, "/panel", nil))
	if err != nil {
		t.Fatalf("GET /panel: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	panel := string(body)
	if response.StatusCode != http.StatusOK ||
		!strings.Contains(panel, "Negocio 9") ||
		!strings.Contains(panel, "Cluckin Bell") ||
		!strings.Contains(panel, "$250,000") {
		t.Fatalf("panel response = %d %q", response.StatusCode, body)
	}

	reviewResponse, err := httpServer.App().Test(httptest.NewRequest(http.MethodPost, "/listings/business/9/review", nil))
	if err != nil {
		t.Fatalf("review request: %v", err)
	}
	_ = reviewResponse.Body.Close()
	if reviewResponse.StatusCode != http.StatusOK || service.Available()[0].Status != review.StatusReviewed {
		t.Fatalf("review status = %d, records = %#v", reviewResponse.StatusCode, service.Available())
	}

	ignoreRequest := httptest.NewRequest(http.MethodPost, "/listings/business/9/ignore", nil)
	ignoreRequest.Header.Set("Accept", "text/html")
	ignoreResponse, err := httpServer.App().Test(ignoreRequest)
	if err != nil {
		t.Fatalf("ignore request: %v", err)
	}
	_ = ignoreResponse.Body.Close()
	if ignoreResponse.StatusCode != http.StatusSeeOther || len(service.Available()) != 0 {
		t.Fatalf("ignore status = %d, records = %#v", ignoreResponse.StatusCode, service.Available())
	}

	listResponse, err := httpServer.App().Test(httptest.NewRequest(http.MethodGet, "/listings", nil))
	if err != nil {
		t.Fatalf("GET /listings: %v", err)
	}
	defer listResponse.Body.Close()
	var listed ListResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil || listed.Count != 0 {
		t.Fatalf("list response = %#v, %v", listed, err)
	}
}
