package houses

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

func TestFallbackSourceUsesAPIFirst(t *testing.T) {
	api := &sourceStub{body: []byte("{\"payload\":[]}")}
	browser := &sourceStub{body: []byte("{\"browser\":true}")}
	source := &FallbackSource{api: api, browser: browser, log: zap.NewNop()}

	body, err := source.FetchMap(context.Background())
	if err != nil || string(body) != "{\"payload\":[]}" {
		t.Fatalf("FetchMap() = %q, %v", body, err)
	}
	if api.calls != 1 || browser.calls != 0 {
		t.Fatalf("calls = api:%d browser:%d", api.calls, browser.calls)
	}
}

func TestFallbackSourceUsesChromeAfterAPIError(t *testing.T) {
	apiErr := errors.New("API unavailable")
	api := &sourceStub{err: apiErr}
	browser := &sourceStub{body: []byte("{\"houses\":[]}")}
	source := &FallbackSource{api: api, browser: browser, log: zap.NewNop()}

	body, err := source.FetchMap(context.Background())
	if err != nil || string(body) != "{\"houses\":[]}" {
		t.Fatalf("FetchMap() = %q, %v", body, err)
	}
	if api.calls != 1 || browser.calls != 1 {
		t.Fatalf("calls = api:%d browser:%d", api.calls, browser.calls)
	}
}

func TestFallbackSourceReturnsBothErrors(t *testing.T) {
	apiErr := errors.New("API unavailable")
	browserErr := errors.New("Chrome unavailable")
	source := &FallbackSource{
		api: &sourceStub{err: apiErr}, browser: &sourceStub{err: browserErr}, log: zap.NewNop(),
	}

	_, err := source.FetchMap(context.Background())
	if !errors.Is(err, apiErr) || !errors.Is(err, browserErr) {
		t.Fatalf("FetchMap() error = %v", err)
	}
}
