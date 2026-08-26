package sarp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestAPIClientFetchBusinesses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/businesses/available" {
			t.Errorf("path = %q", request.URL.Path)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"businesses":[{"id":4,"name":"24/7","price":5000}],"total":1}`)
	}))
	defer server.Close()
	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: server.URL + "/api",
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
	}, zap.NewNop())
	body, err := client.FetchBusinesses(context.Background())
	if err != nil || !strings.Contains(string(body), "businesses") {
		t.Fatalf("FetchBusinesses() = %q, %v", body, err)
	}
}

func TestAPIClientQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/character/search" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.URL.Query().Get("name"); got != "John Doe" {
			t.Errorf("name = %q", got)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer sarp_test" {
			t.Errorf("Authorization = %q", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"payload":[{"id":14381}]}`)
	}))
	defer server.Close()
	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: server.URL + "/api",
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
	}, zap.NewNop())
	body, err := client.Query(context.Background(), "/character/search", url.Values{"name": {"John Doe"}})
	if err != nil || !strings.Contains(string(body), "14381") {
		t.Fatalf("Query() = %q, %v", body, err)
	}
}

func TestAPIClientQueryGTARolProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/businesses/1995" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer sarp_test" {
			t.Errorf("Authorization = %q", got)
		}
		if _, err := request.Cookie("cf_clearance"); err == nil {
			t.Error("GTA Rol request received SARP clearance cookie")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"payload":{"id":1995,"name":"Test"}}`)
	}))
	defer server.Close()
	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: "https://api.sarp.es/api", GTARolAPIBaseURL: server.URL + "/api",
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
	}, zap.NewNop())
	client.storeClearance(&clearance{UserAgent: "clearance-agent", Cookies: []clearanceCookie{{Name: "cf_clearance", Value: "secret"}}})
	body, err := client.QueryProvider(context.Background(), "gta-rol", "/businesses/1995", nil)
	if err != nil || !strings.Contains(string(body), "1995") {
		t.Fatalf("QueryProvider() = %q, %v", body, err)
	}
}

func TestAPIClientQueryRejectsUnknownProvider(t *testing.T) {
	client := NewAPIClient(Config{APIToken: "sarp_test", APITimeout: time.Second}, zap.NewNop())
	if _, err := client.QueryProvider(context.Background(), "external", "/profile", nil); !errors.Is(err, ErrInvalidAPIProvider) {
		t.Fatalf("QueryProvider() error = %v", err)
	}
}

func TestAPIClientQueryRejectsUnsafePaths(t *testing.T) {
	client := NewAPIClient(Config{APIToken: "sarp_test", APIBaseURL: "https://api.sarp.es/api", APITimeout: time.Second}, zap.NewNop())
	for _, path := range []string{"https://example.com/steal", "//example.com/steal", "/../steal", "/%2e%2e/steal", "/map/icons?redirect=https://example.com"} {
		t.Run(path, func(t *testing.T) {
			if _, err := client.Query(context.Background(), path, nil); !errors.Is(err, ErrInvalidAPIPath) {
				t.Fatalf("Query(%q) error = %v", path, err)
			}
		})
	}
}

func TestAPIClientQueryDoesNotFollowRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		destinationCalls.Add(1)
		_, _ = fmt.Fprint(response, `{}`)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Location", destination.URL)
		response.WriteHeader(http.StatusFound)
	}))
	defer source.Close()
	client := NewAPIClient(Config{
		APIToken: "sarp_test", APIBaseURL: source.URL,
		APITimeout: time.Second, APIUserAgent: "test-client/1.0",
	}, zap.NewNop())
	if _, err := client.Query(context.Background(), "/redirect", nil); err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("Query() error = %v", err)
	}
	if destinationCalls.Load() != 0 {
		t.Fatalf("redirect destination calls = %d", destinationCalls.Load())
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
