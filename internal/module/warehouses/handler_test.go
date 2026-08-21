package warehouses

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/zap"
)

func TestWarehouseRoutes(t *testing.T) {
	store, err := review.NewStore(review.Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	listingService := listings.NewService(store, disabledNotifier{}, zap.NewNop())
	service := NewService(&sourceStub{body: []byte(`{"payload":{"warehouses":[{"id":4,"address":"Market","value":5000}]}}`)}, NewMemoryRepository(), listingService, zap.NewNop())
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(service))
	refresh, err := httpServer.App().Test(httptest.NewRequest(http.MethodPost, "/warehouses/refresh", nil))
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	_ = refresh.Body.Close()
	response, err := httpServer.App().Test(httptest.NewRequest(http.MethodGet, "/warehouses", nil))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer response.Body.Close()
	var snapshot Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil || snapshot.Count != 1 {
		t.Fatalf("snapshot = %#v, %v", snapshot, err)
	}
}
