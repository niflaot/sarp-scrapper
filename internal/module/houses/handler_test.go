package houses

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/zap"
)

func TestHouseRoutes(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(&sourceStub{body: []byte(`[{"key":749,"address":"Rodeo","price":100,"available":true}]`)}, repository, zap.NewNop())
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(service))

	refreshResponse, err := httpServer.App().Test(newRequest(t, http.MethodPost, "/houses/refresh"))
	if err != nil {
		t.Fatalf("POST /houses/refresh: %v", err)
	}
	_ = refreshResponse.Body.Close()
	if refreshResponse.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d", refreshResponse.StatusCode)
	}

	response, err := httpServer.App().Test(newRequest(t, http.MethodGet, "/houses"))
	if err != nil {
		t.Fatalf("GET /houses: %v", err)
	}
	defer response.Body.Close()
	var snapshot Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.StatusCode != http.StatusOK || snapshot.Count != 1 || snapshot.Houses[0].Key != "749" {
		t.Fatalf("unexpected response: %#v", snapshot)
	}
}

func TestRefreshRouteWhenDisabled(t *testing.T) {
	service := NewService(&sourceStub{err: sarp.ErrDisabled}, NewMemoryRepository(), zap.NewNop())
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(service))
	response, err := httpServer.App().Test(newRequest(t, http.MethodPost, "/houses/refresh"))
	if err != nil {
		t.Fatalf("POST /houses/refresh: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func newRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	return request
}
