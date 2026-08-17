package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

type authenticatorStub struct {
	loginState     session.State
	loginErr       error
	twoFactorState session.State
	twoFactorErr   error
	loginCode      string
	twoFactorCode  string
}

func (a *authenticatorStub) Login(_ context.Context, code string) (session.State, error) {
	a.loginCode = code
	return a.loginState, a.loginErr
}
func (a *authenticatorStub) SubmitTwoFactor(_ context.Context, code string) (session.State, error) {
	a.twoFactorCode = code
	return a.twoFactorState, a.twoFactorErr
}

func TestAuthRoutes(t *testing.T) {
	stub := &authenticatorStub{loginState: session.StatePending, twoFactorState: session.StateAvailable}
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(newService(stub)))

	response := authRequest(t, httpServer, "/auth/login", `{"twoFactorCode":"123456"}`)
	if response.StatusCode != http.StatusAccepted || stub.loginCode != "123456" {
		t.Fatalf("login status/code = %d/%q", response.StatusCode, stub.loginCode)
	}
	_ = response.Body.Close()
	response = authRequest(t, httpServer, "/auth/2fa", `{"code":"654321"}`)
	if response.StatusCode != http.StatusOK || stub.twoFactorCode != "654321" {
		t.Fatalf("2FA status/code = %d/%q", response.StatusCode, stub.twoFactorCode)
	}
	_ = response.Body.Close()
}

func TestAuthRoutesExposeRequestBodiesInOpenAPI(t *testing.T) {
	stub := &authenticatorStub{}
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(newService(stub)))
	request, err := http.NewRequest(http.MethodGet, "/openapi.json", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
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
	for _, path := range []string{"/auth/login", "/auth/2fa"} {
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI path %s missing", path)
		}
		post, ok := pathItem["post"].(map[string]any)
		if !ok || post["requestBody"] == nil {
			t.Fatalf("OpenAPI request body missing for %s: %#v", path, post)
		}
	}
}

func TestAuthRouteErrors(t *testing.T) {
	tests := []struct {
		name, path, body string
		err              error
		want             int
	}{
		{name: "disabled", path: "/auth/login", body: `{}`, err: sarp.ErrDisabled, want: http.StatusServiceUnavailable},
		{name: "no pending challenge", path: "/auth/2fa", body: `{"code":"123456"}`, err: sarp.ErrNoPendingTwoFactor, want: http.StatusConflict},
		{name: "invalid code", path: "/auth/2fa", body: `{"code":"123456"}`, err: sarp.ErrInvalidTwoFactorCode, want: http.StatusBadRequest},
		{name: "missing code", path: "/auth/2fa", body: `{}`, want: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &authenticatorStub{loginErr: test.err, twoFactorErr: test.err}
			httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
			Register(httpServer, NewHandler(newService(stub)))
			response := authRequest(t, httpServer, test.path, test.body)
			_ = response.Body.Close()
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
		})
	}
}

func authRequest(t *testing.T, httpServer *server.Server, path, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.App().Test(request)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return response
}
