package sarp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

func TestAPIClientFetchMap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/map/icons" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer sarp_test" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if got := request.Header.Get("User-Agent"); got != "test-client/1.0" {
			t.Errorf("User-Agent = %q", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, "{\"message\":\"OK\",\"payload\":[{\"key\":749,\"address\":\"Rodeo\",\"price\":100,\"icon\":\"radar_propertyG.png\"}]}")
	}))
	defer server.Close()

	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: server.URL + "/api",
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
	}, zap.NewNop())
	body, err := client.FetchMap(context.Background())
	if err != nil || !strings.Contains(string(body), "radar_propertyG") {
		t.Fatalf("FetchMap() = %q, %v", body, err)
	}
	if client.Status() != session.StateAvailable {
		t.Fatalf("Status() = %q", client.Status())
	}
}

func TestAPIClientErrors(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		client := NewAPIClient(Config{APITimeout: time.Second}, zap.NewNop())
		if _, err := client.FetchMap(context.Background()); !errors.Is(err, ErrAPITokenMissing) {
			t.Fatalf("FetchMap() error = %v", err)
		}
		if client.Status() != session.StatePending {
			t.Fatalf("Status() = %q", client.Status())
		}
	})

	t.Run("Cloudflare challenge", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("cf-mitigated", "challenge")
			response.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(response, `<html>challenge</html>`)
		}))
		defer server.Close()
		client := NewAPIClient(Config{APIToken: "sarp_test", APIBaseURL: server.URL, APITimeout: time.Second}, zap.NewNop())
		if _, err := client.FetchMap(context.Background()); !errors.Is(err, ErrAPICloudflareChallenge) {
			t.Fatalf("FetchMap() error = %v", err)
		}
	})

	t.Run("non JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(response, `not json`)
		}))
		defer server.Close()
		client := NewAPIClient(Config{APIToken: "sarp_test", APIBaseURL: server.URL, APITimeout: time.Second}, zap.NewNop())
		if _, err := client.FetchMap(context.Background()); err == nil || !strings.Contains(err.Error(), "non-JSON") {
			t.Fatalf("FetchMap() error = %v", err)
		}
	})
}

func TestAPIClientUsesFlareSolverrClearance(t *testing.T) {
	var apiCalls atomic.Int32
	apiServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		apiCalls.Add(1)
		if got := request.Header.Get("Authorization"); got != "Bearer sarp_test" {
			t.Errorf("Authorization = %q", got)
		}
		cookie, err := request.Cookie("cf_clearance")
		if err != nil || cookie.Value != "clearance_test" || request.UserAgent() != "flare-agent/1.0" {
			response.Header().Set("cf-mitigated", "challenge")
			response.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(response, `<html>challenge</html>`)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"payload":[{"key":749,"icon":"radar_propertyG.png"}]}`)
	}))
	defer apiServer.Close()

	var flareCalls atomic.Int32
	flareServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		flareCalls.Add(1)
		var payload flareSolverrRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode FlareSolverr request: %v", err)
		}
		if payload.Command != "request.get" || payload.URL != apiServer.URL+"/map/icons" || !payload.ReturnOnlyCookies {
			t.Errorf("FlareSolverr payload = %#v", payload)
		}
		if strings.Contains(payload.URL, "sarp_test") {
			t.Fatal("API token leaked to FlareSolverr")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"status":"ok","message":"Challenge solved!","solution":{"cookies":[{"name":"cf_clearance","value":"clearance_test"}],"userAgent":"flare-agent/1.0"}}`)
	}))
	defer flareServer.Close()

	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: apiServer.URL,
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
		FlareSolverrURL: flareServer.URL, FlareSolverrTimeout: time.Second,
	}, zap.NewNop())
	for attempt := 0; attempt < 2; attempt++ {
		body, err := client.FetchMap(context.Background())
		if err != nil || !strings.Contains(string(body), "radar_propertyG") {
			t.Fatalf("FetchMap() attempt %d = %q, %v", attempt, body, err)
		}
	}
	if got := flareCalls.Load(); got != 1 {
		t.Fatalf("FlareSolverr calls = %d, want 1", got)
	}
	if got := apiCalls.Load(); got != 3 {
		t.Fatalf("API calls = %d, want 3", got)
	}
}

func TestAPIClientReportsFlareSolverrFailure(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("cf-mitigated", "challenge")
		response.WriteHeader(http.StatusForbidden)
	}))
	defer apiServer.Close()
	flareServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"status":"error","message":"solve failed"}`)
	}))
	defer flareServer.Close()

	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: apiServer.URL,
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
		FlareSolverrURL: flareServer.URL, FlareSolverrTimeout: time.Second,
	}, zap.NewNop())
	_, err := client.FetchMap(context.Background())
	if !errors.Is(err, ErrAPICloudflareChallenge) || !errors.Is(err, ErrFlareSolverrFailed) {
		t.Fatalf("FetchMap() error = %v", err)
	}
}
