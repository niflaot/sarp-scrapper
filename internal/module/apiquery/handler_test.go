package apiquery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/zap"
)

type clientStub struct {
	path  string
	query url.Values
	body  []byte
	err   error
}

func (c *clientStub) Query(_ context.Context, path string, query url.Values) ([]byte, error) {
	c.path, c.query = path, query
	return c.body, c.err
}

func TestQueryRoute(t *testing.T) {
	client := &clientStub{body: []byte(`{"payload":[{"id":14381}]}`)}
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(NewService(client)))
	request, err := http.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"path":"/character/search","query":{"name":["John Doe"]}}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.App().Test(request)
	if err != nil {
		t.Fatalf("POST /api/query: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || client.path != "/character/search" || client.query.Get("name") != "John Doe" {
		t.Fatalf("status/path/query = %d/%q/%q", response.StatusCode, client.path, client.query.Get("name"))
	}
}

func TestQueryRouteErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		err  error
		want int
	}{
		{name: "invalid body", body: `{`, want: http.StatusBadRequest},
		{name: "missing path", body: `{}`, want: http.StatusBadRequest},
		{name: "unsafe path", body: `{"path":"https://example.com"}`, err: sarp.ErrInvalidAPIPath, want: http.StatusBadRequest},
		{name: "missing token", body: `{"path":"/account/profile"}`, err: sarp.ErrAPITokenMissing, want: http.StatusServiceUnavailable},
		{name: "upstream", body: `{"path":"/account/profile"}`, err: errors.New("upstream failed"), want: http.StatusBadGateway},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &clientStub{err: test.err}
			httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
			Register(httpServer, NewHandler(NewService(client)))
			request, _ := http.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/json")
			response, err := httpServer.App().Test(request)
			if err != nil {
				t.Fatalf("POST /api/query: %v", err)
			}
			_ = response.Body.Close()
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
		})
	}
}

func TestQueryRouteExposesRequestBodyInOpenAPI(t *testing.T) {
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(NewService(&clientStub{})))
	request, _ := http.NewRequest(http.MethodGet, "/openapi.json", nil)
	response, err := httpServer.App().Test(request)
	if err != nil {
		t.Fatalf("GET /openapi.json: %v", err)
	}
	defer response.Body.Close()
	var document map[string]any
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI: %v", err)
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI paths missing: %#v", document)
	}
	path, ok := paths["/api/query"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI query path missing: %#v", paths)
	}
	post, ok := path["post"].(map[string]any)
	if !ok || post["requestBody"] == nil {
		t.Fatalf("OpenAPI request body missing: %#v", path)
	}
}
