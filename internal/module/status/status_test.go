package status

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

func TestRegister(t *testing.T) {
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, statusProvider{state: session.StatePending})
	request, _ := http.NewRequest(http.MethodGet, "/status", nil)
	response, err := httpServer.App().Test(request)
	if err != nil {
		t.Fatalf("GET /status: %v", err)
	}
	defer response.Body.Close()
	var body Response
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.StatusCode != http.StatusOK || body.Status != session.StatePending {
		t.Fatalf("unexpected response: %#v", body)
	}
}

type statusProvider struct{ state session.State }

func (p statusProvider) Status() session.State { return p.state }
