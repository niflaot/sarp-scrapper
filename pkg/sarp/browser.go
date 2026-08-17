package sarp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrDisabled             = errors.New("SARP browser is disabled")
	ErrSessionPending       = errors.New("SARP session is pending")
	ErrNoPendingTwoFactor   = errors.New("no SARP 2FA challenge is pending")
	ErrInvalidTwoFactorCode = errors.New("2FA code must contain 4 to 10 digits")
	ErrBrowserChallenge     = errors.New("SARP presented a browser challenge that requires manual intervention")
)

var twoFactorPattern = regexp.MustCompile(`^[0-9]{4,10}$`)
var popupPattern = regexp.MustCompile(`(?is)Llave\s+([0-9]+)\s+(.+?)\s+\$([0-9.,]+)`)

type Browser struct {
	config Config
	log    *zap.Logger

	operationMu   sync.Mutex
	stateMu       sync.RWMutex
	state         session.State
	allocator     context.Context
	browser       context.Context
	cancelAlloc   context.CancelFunc
	cancelBrowser context.CancelFunc
}

func NewBrowser(config Config, log *zap.Logger) *Browser {
	return &Browser{config: config, log: log, state: session.StatePending}
}

func RegisterLifecycle(lifecycle fx.Lifecycle, browser *Browser) {
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error { browser.Close(); return nil }})
}

func (b *Browser) Status() session.State {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	return b.state
}

func (b *Browser) Login(ctx context.Context, twoFactorCode string) (session.State, error) {
	b.operationMu.Lock()
	defer b.operationMu.Unlock()
	if !b.config.Enabled {
		return session.StatePending, ErrDisabled
	}
	if twoFactorCode != "" && !twoFactorPattern.MatchString(twoFactorCode) {
		return session.StatePending, ErrInvalidTwoFactorCode
	}
	if err := b.ensureStarted(); err != nil {
		return session.StatePending, err
	}

	opCtx, cancel := b.operationContext(ctx, b.config.LoginTimeout)
	defer cancel()
	if err := chromedp.Run(opCtx, chromedp.Navigate(b.config.LoginURL), chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		return b.fail(fmt.Errorf("open SARP login: %w", err))
	}
	phase, err := b.waitForInitialPhase(opCtx)
	if err != nil {
		return b.fail(err)
	}
	for {
		switch phase {
		case phaseAvailable:
			return b.succeed(), nil
		case phaseChallenge:
			phase, err = b.waitForChallengeResolution(opCtx)
			if err != nil {
				return b.fail(err)
			}
		case phaseLogin:
			if err := chromedp.Run(opCtx,
				chromedp.SetValue(b.config.EmailSelector, b.config.Email, chromedp.ByQuery),
				chromedp.SetValue(b.config.PasswordSelector, b.config.Password, chromedp.ByQuery),
				chromedp.Click(b.config.SubmitSelector, chromedp.ByQuery),
			); err != nil {
				return b.fail(fmt.Errorf("submit SARP credentials: %w", err))
			}
			phase, err = b.waitForLoginResult(opCtx)
			if err != nil {
				return b.fail(err)
			}
		case phaseTwoFactor:
			goto twoFactor
		default:
			return b.fail(fmt.Errorf("SARP login did not reach an authenticated or 2FA state"))
		}
	}

twoFactor:
	b.setState(session.StatePending)
	if twoFactorCode == "" {
		return session.StatePending, nil
	}
	return b.submitTwoFactorLocked(opCtx, twoFactorCode)
}

func (b *Browser) SubmitTwoFactor(ctx context.Context, code string) (session.State, error) {
	b.operationMu.Lock()
	defer b.operationMu.Unlock()
	if !b.config.Enabled {
		return session.StatePending, ErrDisabled
	}
	if !twoFactorPattern.MatchString(code) {
		return session.StatePending, ErrInvalidTwoFactorCode
	}
	if b.Status() != session.StatePending || b.browser == nil {
		return session.StatePending, ErrNoPendingTwoFactor
	}
	opCtx, cancel := b.operationContext(ctx, b.config.LoginTimeout)
	defer cancel()
	phase, err := b.detectPhase(opCtx)
	if err != nil {
		return b.fail(err)
	}
	if phase != phaseTwoFactor {
		return b.fail(ErrNoPendingTwoFactor)
	}
	return b.submitTwoFactorLocked(opCtx, code)
}

func (b *Browser) submitTwoFactorLocked(ctx context.Context, code string) (session.State, error) {
	if err := chromedp.Run(ctx,
		chromedp.SetValue(b.config.TwoFactorSelector, code, chromedp.ByQuery),
		chromedp.Click(b.config.TwoFactorSubmitSelector, chromedp.ByQuery),
	); err != nil {
		return b.fail(fmt.Errorf("submit SARP 2FA code: %w", err))
	}
	phase, err := b.waitForLoginResult(ctx)
	if err != nil {
		return b.fail(err)
	}
	if phase == phaseChallenge {
		phase, err = b.waitForChallengeResolution(ctx)
		if err != nil {
			return b.fail(err)
		}
	}
	if phase == phaseAvailable {
		return b.succeed(), nil
	}
	return b.fail(fmt.Errorf("SARP rejected the 2FA code or kept the challenge pending"))
}

func (b *Browser) FetchMap(ctx context.Context) ([]byte, error) {
	b.operationMu.Lock()
	defer b.operationMu.Unlock()
	if !b.config.Enabled {
		return nil, ErrDisabled
	}
	if b.browser == nil {
		if err := b.ensureStarted(); err != nil {
			return nil, err
		}
	}
	if b.browser == nil {
		return nil, ErrSessionPending
	}

	tabCtx, cancelTab := chromedp.NewContext(b.browser)
	defer cancelTab()
	opCtx, cancel := context.WithTimeout(tabCtx, b.config.LoginTimeout+b.config.MapSettleTime)
	stop := context.AfterFunc(ctx, cancel)
	defer func() { stop(); cancel() }()

	var captureMu sync.Mutex
	var responseBodies []json.RawMessage
	var captureWG sync.WaitGroup
	capturing := true
	pendingResponses := make(map[network.RequestID]bool)
	chromedp.ListenTarget(opCtx, func(event any) {
		switch typed := event.(type) {
		case *network.EventResponseReceived:
			if typed.Type != network.ResourceTypeXHR && typed.Type != network.ResourceTypeFetch {
				return
			}
			captureMu.Lock()
			if !capturing {
				captureMu.Unlock()
				return
			}
			pendingResponses[typed.RequestID] = true
			captureMu.Unlock()
		case *network.EventLoadingFinished:
			captureMu.Lock()
			if !capturing || !pendingResponses[typed.RequestID] {
				captureMu.Unlock()
				return
			}
			delete(pendingResponses, typed.RequestID)
			captureWG.Add(1)
			captureMu.Unlock()
			go func(requestID network.RequestID) {
				defer captureWG.Done()
				browserContext := chromedp.FromContext(opCtx)
				if browserContext == nil || browserContext.Target == nil {
					return
				}
				executorContext := cdp.WithExecutor(opCtx, browserContext.Target)
				body, err := network.GetResponseBody(requestID).Do(executorContext)
				if err == nil && json.Valid(body) {
					captureMu.Lock()
					responseBodies = append(responseBodies, append(json.RawMessage(nil), body...))
					captureMu.Unlock()
				} else if err != nil {
					b.log.Debug("could not read browser response body", zap.Error(err))
				}
			}(typed.RequestID)
		case *network.EventWebSocketFrameReceived:
			body := []byte(typed.Response.PayloadData)
			if json.Valid(body) {
				captureMu.Lock()
				if capturing {
					responseBodies = append(responseBodies, append(json.RawMessage(nil), body...))
				}
				captureMu.Unlock()
			}
		}
	})

	if err := chromedp.Run(opCtx, network.Enable(), chromedp.Navigate(b.config.MapURL), chromedp.WaitReady("body", chromedp.ByQuery), chromedp.Sleep(b.config.MapSettleTime)); err != nil {
		return nil, fmt.Errorf("load SARP map: %w", err)
	}
	phase, phaseErr := b.detectPhase(opCtx)
	var sessionErr error
	if phaseErr == nil && phase != phaseAvailable {
		initialPhase := phase
		b.setState(session.StatePending)
		b.log.Debug("SARP map requires manual browser authentication", zap.String("phase", phase.String()))
		phase, phaseErr = b.waitForPhase(opCtx, "wait for manual SARP browser authentication", phaseAvailable)
		if phaseErr != nil {
			if initialPhase == phaseChallenge {
				sessionErr = fmt.Errorf("%w: %v", ErrBrowserChallenge, phaseErr)
			} else {
				sessionErr = fmt.Errorf("%w: %v", ErrSessionPending, phaseErr)
			}
		} else if err := chromedp.Run(opCtx,
			network.Enable(),
			chromedp.Navigate(b.config.MapURL),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Sleep(b.config.MapSettleTime),
		); err != nil {
			sessionErr = fmt.Errorf("reload SARP map after manual authentication: %w", err)
		}
	}
	var popupTexts []string
	var location string
	var markerCount int
	if sessionErr == nil {
		if err := chromedp.Run(opCtx,
			chromedp.Location(&location),
			chromedp.Evaluate(`document.querySelectorAll('img.leaflet-marker-icon[src*="radar_propertyG"]').length`, &markerCount),
			chromedp.Evaluate(popupCollectionScript, &popupTexts, func(params *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
				return params.WithAwaitPromise(true)
			}),
		); err != nil {
			b.log.Debug("could not collect Leaflet house popups", zap.Error(err))
		}
	}
	captureMu.Lock()
	capturing = false
	captureMu.Unlock()
	captureWG.Wait()
	if sessionErr != nil {
		return nil, sessionErr
	}
	b.setState(session.StateAvailable)
	b.log.Debug("SARP map capture completed", zap.String("url", location), zap.Int("house_markers", markerCount), zap.Int("json_responses", len(responseBodies)), zap.Int("popup_records", len(popupTexts)))

	payload := struct {
		Responses []json.RawMessage `json:"responses"`
		Houses    []map[string]any  `json:"houses"`
	}{Responses: responseBodies, Houses: popupHouses(popupTexts)}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode captured SARP map data: %w", err)
	}
	return body, nil
}

func (b *Browser) Close() {
	b.operationMu.Lock()
	defer b.operationMu.Unlock()
	if b.cancelBrowser != nil {
		b.cancelBrowser()
	}
	if b.cancelAlloc != nil {
		b.cancelAlloc()
	}
	b.stateMu.Lock()
	b.browser, b.allocator, b.cancelBrowser, b.cancelAlloc = nil, nil, nil, nil
	b.state = session.StatePending
	b.stateMu.Unlock()
}

func (b *Browser) ensureStarted() error {
	if b.browser != nil {
		return nil
	}
	if b.config.ChromeRemoteDebugURL != "" {
		allocator, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), b.config.ChromeRemoteDebugURL)
		browser, cancelBrowser := chromedp.NewContext(allocator)
		if err := chromedp.Run(browser); err != nil {
			cancelBrowser()
			cancelAlloc()
			return fmt.Errorf("attach to Chrome session: %w", err)
		}
		b.allocator, b.browser, b.cancelAlloc, b.cancelBrowser = allocator, browser, cancelAlloc, cancelBrowser
		b.log.Debug("attached to existing local Chrome session")
		return nil
	}
	profileDir, err := filepath.Abs(b.config.ProfileDir)
	if err != nil {
		return fmt.Errorf("resolve SARP browser profile: %w", err)
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return fmt.Errorf("create SARP browser profile: %w", err)
	}
	if err := os.Chmod(profileDir, 0o700); err != nil {
		return fmt.Errorf("secure SARP browser profile: %w", err)
	}
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.UserDataDir(profileDir), chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck, chromedp.Flag("headless", b.config.Headless))
	if b.config.ChromePath != "" {
		options = append(options, chromedp.ExecPath(b.config.ChromePath))
	}
	allocator, cancelAlloc := chromedp.NewExecAllocator(context.Background(), options...)
	browser, cancelBrowser := chromedp.NewContext(allocator)
	if err := chromedp.Run(browser); err != nil {
		cancelBrowser()
		cancelAlloc()
		return fmt.Errorf("start isolated Chrome browser: %w", err)
	}
	b.allocator, b.browser, b.cancelAlloc, b.cancelBrowser = allocator, browser, cancelAlloc, cancelBrowser
	return nil
}

func (b *Browser) operationContext(request context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(b.browser, timeout)
	stop := context.AfterFunc(request, cancel)
	return ctx, func() { stop(); cancel() }
}

type loginPhase int

const (
	phaseUnknown loginPhase = iota
	phaseLogin
	phaseTwoFactor
	phaseAvailable
	phaseChallenge
)

func (p loginPhase) String() string {
	switch p {
	case phaseLogin:
		return "login"
	case phaseTwoFactor:
		return "two_factor"
	case phaseAvailable:
		return "available"
	case phaseChallenge:
		return "challenge"
	default:
		return "unknown"
	}
}

func (b *Browser) detectPhase(ctx context.Context) (loginPhase, error) {
	var location string
	var flags struct{ Login, TwoFactor, Challenge bool }
	script := fmt.Sprintf(`(() => {
      const exists = (selector) => { try { return !!document.querySelector(selector); } catch (_) { return false; } };
      const text = (document.body && document.body.innerText || '').toLowerCase();
	  const title = (document.title || '').toLowerCase();
	  const cloudflare = typeof window._cf_chl_opt !== 'undefined' || exists('#challenge-error-text,script[src*="/cdn-cgi/challenge-platform/"]') || title.includes('just a moment');
      return {login: exists(%s) && exists(%s), twoFactor: exists(%s), challenge: cloudflare || exists('iframe[src*="captcha"],iframe[src*="challenge"],.cf-challenge') || text.includes('verify you are human') || text.includes('verifica que eres humano')};
    })()`, strconv.Quote(b.config.EmailSelector), strconv.Quote(b.config.PasswordSelector), strconv.Quote(b.config.TwoFactorSelector))
	if err := chromedp.Run(ctx, chromedp.Location(&location), chromedp.Evaluate(script, &flags)); err != nil {
		return phaseUnknown, fmt.Errorf("inspect SARP login state: %w", err)
	}
	if flags.Challenge {
		return phaseChallenge, nil
	}
	if flags.TwoFactor {
		return phaseTwoFactor, nil
	}
	if strings.Contains(location, "/app/ucp/") && !flags.Login {
		return phaseAvailable, nil
	}
	if flags.Login {
		return phaseLogin, nil
	}
	return phaseUnknown, nil
}

func (b *Browser) waitForLoginResult(ctx context.Context) (loginPhase, error) {
	return b.waitForPhase(ctx, "wait for SARP login result", phaseAvailable, phaseTwoFactor, phaseChallenge)
}

func (b *Browser) waitForInitialPhase(ctx context.Context) (loginPhase, error) {
	return b.waitForPhase(ctx, "wait for SARP login page", phaseLogin, phaseTwoFactor, phaseAvailable, phaseChallenge)
}

func (b *Browser) waitForChallengeResolution(ctx context.Context) (loginPhase, error) {
	b.log.Debug("SARP browser challenge is waiting for manual completion")
	phase, err := b.waitForPhase(ctx, "wait for manual SARP browser challenge", phaseLogin, phaseTwoFactor, phaseAvailable)
	if err != nil {
		return phaseUnknown, fmt.Errorf("%w: %v", ErrBrowserChallenge, err)
	}
	return phase, nil
}

func (b *Browser) waitForPhase(ctx context.Context, operation string, accepted ...loginPhase) (loginPhase, error) {
	acceptedPhases := make(map[loginPhase]struct{}, len(accepted))
	for _, phase := range accepted {
		acceptedPhases[phase] = struct{}{}
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		phase, err := b.detectPhase(ctx)
		if err == nil {
			if _, ok := acceptedPhases[phase]; ok {
				return phase, nil
			}
		}
		select {
		case <-ctx.Done():
			return phaseUnknown, fmt.Errorf("%s: %w", operation, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (b *Browser) setState(state session.State) {
	b.stateMu.Lock()
	b.state = state
	b.stateMu.Unlock()
}
func (b *Browser) succeed() session.State {
	b.setState(session.StateAvailable)
	return session.StateAvailable
}
func (b *Browser) fail(err error) (session.State, error) {
	b.setState(session.StatePending)
	return session.StatePending, err
}

const popupCollectionScript = `(async () => {
  const markers = Array.from(document.querySelectorAll('img.leaflet-marker-icon[src*="radar_propertyG"]'));
  const results = [];
  for (const marker of markers) {
    marker.click();
    await new Promise(resolve => setTimeout(resolve, 75));
    const popup = document.querySelector('.leaflet-popup-content');
    if (popup && popup.innerText) results.push(popup.innerText);
  }
  return results;
})()`

func popupHouses(texts []string) []map[string]any {
	result := make([]map[string]any, 0, len(texts))
	seen := make(map[string]bool)
	for _, text := range texts {
		match := popupPattern.FindStringSubmatch(text)
		if len(match) != 4 || seen[match[1]] {
			continue
		}
		priceText := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, match[3])
		price, err := strconv.ParseInt(priceText, 10, 64)
		if err != nil {
			continue
		}
		address := strings.Join(strings.Fields(match[2]), " ")
		result = append(result, map[string]any{"key": match[1], "address": address, "price": price, "available": true, "type": "house"})
		seen[match[1]] = true
	}
	return result
}
