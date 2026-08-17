package sarp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

func TestPopupHouses(t *testing.T) {
	houses := popupHouses([]string{
		"Llave 749\nOlympia Avenue, Rodeo\n$1,338,876",
		"Llave 749\nDuplicate\n$1",
		"not a house",
	})
	if len(houses) != 1 || houses[0]["key"] != "749" || houses[0]["price"] != int64(1338876) {
		t.Fatalf("unexpected houses: %#v", houses)
	}
}

func TestBrowserDisabledAndValidation(t *testing.T) {
	browser := NewBrowser(Config{}, zap.NewNop())
	if browser.Status() != session.StatePending {
		t.Fatalf("Status() = %q", browser.Status())
	}
	if _, err := browser.Login(context.Background(), ""); err != ErrDisabled {
		t.Fatalf("Login() error = %v", err)
	}
	if _, err := browser.SubmitTwoFactor(context.Background(), "abc"); err != ErrDisabled {
		t.Fatalf("SubmitTwoFactor() error = %v", err)
	}
	if _, err := browser.FetchMap(context.Background()); err != ErrDisabled {
		t.Fatalf("FetchMap() error = %v", err)
	}
	browser.Close()
}

func TestBrowserLoginTwoFactorAndMapCapture(t *testing.T) {
	server := newFakeUCP(t)
	config := validConfig()
	config.LoginURL = server.URL + "/"
	config.MapURL = server.URL + "/app/ucp/map"
	config.ProfileDir = t.TempDir()
	config.Headless = true
	config.LoginTimeout = 8 * time.Second
	config.MapSettleTime = 500 * time.Millisecond
	browser := NewBrowser(config, zap.NewNop())
	t.Cleanup(browser.Close)

	state, err := browser.Login(context.Background(), "")
	if err != nil || state != session.StatePending {
		t.Fatalf("Login() = %q, %v", state, err)
	}
	state, err = browser.SubmitTwoFactor(context.Background(), "123456")
	if err != nil || state != session.StateAvailable {
		t.Fatalf("SubmitTwoFactor() = %q, %v", state, err)
	}

	body, err := browser.FetchMap(context.Background())
	if err != nil {
		t.Fatalf("FetchMap() error = %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `"key":501`) || !strings.Contains(text, `"key":"749"`) {
		t.Fatalf("captured map data missing houses: %s", text)
	}
}

func TestBrowserDetectsCloudflareChallenge(t *testing.T) {
	server := newFakeUCP(t)
	config := validConfig()
	config.LoginURL = server.URL + "/challenge"
	config.ProfileDir = t.TempDir()
	config.Headless = true
	config.LoginTimeout = time.Second
	browser := NewBrowser(config, zap.NewNop())
	t.Cleanup(browser.Close)

	state, err := browser.Login(context.Background(), "")
	if state != session.StatePending || !errors.Is(err, ErrBrowserChallenge) {
		t.Fatalf("Login() = %q, %v", state, err)
	}
}

func TestBrowserContinuesAfterManualChallenge(t *testing.T) {
	server := newFakeUCP(t)
	config := validConfig()
	config.LoginURL = server.URL + "/challenge-resolves"
	config.ProfileDir = t.TempDir()
	config.Headless = true
	config.LoginTimeout = 8 * time.Second
	browser := NewBrowser(config, zap.NewNop())
	t.Cleanup(browser.Close)

	state, err := browser.Login(context.Background(), "")
	if err != nil || state != session.StatePending {
		t.Fatalf("Login() = %q, %v", state, err)
	}
}

func TestBrowserFetchMapWaitsForManualAuthentication(t *testing.T) {
	server := newFakeUCP(t)
	config := validConfig()
	config.MapURL = server.URL + "/app/ucp/manual-map"
	config.ProfileDir = t.TempDir()
	config.Headless = true
	config.LoginTimeout = 8 * time.Second
	config.MapSettleTime = 300 * time.Millisecond
	browser := NewBrowser(config, zap.NewNop())
	t.Cleanup(browser.Close)

	body, err := browser.FetchMap(context.Background())
	if err != nil {
		t.Fatalf("FetchMap() error = %v", err)
	}
	if !strings.Contains(string(body), `"key":902`) {
		t.Fatalf("FetchMap() body = %s", body)
	}
	if browser.Status() != session.StateAvailable {
		t.Fatalf("Status() = %q", browser.Status())
	}
}

func newFakeUCP(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><body><div id="app">Loading</div><script>setTimeout(()=>{document.querySelector('#app').innerHTML='<form id="login"><input type="email"><input type="password"><button type="submit">Login</button></form>';document.querySelector('form').addEventListener('submit',event=>{event.preventDefault();location.href='/2fa';});},350);</script></body></html>`)
	})
	mux.HandleFunc("/2fa", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><body><form><input name="code" autocomplete="one-time-code"><button type="submit">Verify</button></form><script>document.querySelector('form').addEventListener('submit',event=>{event.preventDefault();if(document.querySelector('input').value==='123456')location.href='/app/ucp/home';});</script></body></html>`)
	})
	mux.HandleFunc("/challenge", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><head><title>Just a moment...</title></head><body><div id="challenge-error-text"></div><script>window._cf_chl_opt={};</script></body></html>`)
	})
	mux.HandleFunc("/challenge-resolves", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><head><title>Just a moment...</title></head><body><div id="challenge-error-text"></div><script>window._cf_chl_opt={};setTimeout(()=>{delete window._cf_chl_opt;document.title='SARP';document.body.innerHTML='<form><input type="email"><input type="password"><button type="submit">Login</button></form>';document.querySelector('form').addEventListener('submit',event=>{event.preventDefault();location.href='/2fa';});},350);</script></body></html>`)
	})
	mux.HandleFunc("/app/ucp/home", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><body>Authenticated</body></html>`)
	})
	mux.HandleFunc("/app/ucp/map", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><body><img class="leaflet-marker-icon" src="radar_propertyG.png"><script>
fetch('/map-data');
document.querySelector('img').addEventListener('click',()=>{const popup=document.createElement('div');popup.className='leaflet-popup-content';popup.innerText='Llave 749\nOlympia Avenue, Rodeo\n$1,338,876';document.body.appendChild(popup);});
</script></body></html>`)
	})
	mux.HandleFunc("/app/ucp/manual-map", func(response http.ResponseWriter, request *http.Request) {
		if cookie, err := request.Cookie("manual-auth"); err != nil || cookie.Value != "yes" {
			http.Redirect(response, request, "/manual-login", http.StatusFound)
			return
		}
		_, _ = fmt.Fprint(response, `<html><body><script>fetch('/manual-map-data');</script></body></html>`)
	})
	mux.HandleFunc("/manual-login", func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, `<html><body><form><input type="email"><input type="password"><button type="submit">Login</button></form><script>setTimeout(()=>{document.cookie='manual-auth=yes; path=/';location.href='/app/ucp/manual-map';},350);</script></body></html>`)
	})
	mux.HandleFunc("/manual-map-data", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, "{\"houses\":[{\"key\":902,\"address\":\"Manual Login\",\"price\":300,\"available\":true}]}")
	})
	mux.HandleFunc("/map-data", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"houses":[{"key":501,"address":"Market","price":200,"available":true}]}`)
	})
	return httptest.NewServer(mux)
}
