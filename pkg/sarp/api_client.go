package sarp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/zap"
)

var (
	ErrAPITokenMissing        = errors.New("SARP API token is not configured")
	ErrAPICloudflareChallenge = errors.New("SARP API request was blocked by a Cloudflare challenge")
	ErrFlareSolverrFailed     = errors.New("FlareSolverr could not establish SARP clearance")
)

const maxAPIResponseSize = 20 << 20
const maxFlareSolverrResponseSize = 2 << 20

type flareSolverrRequest struct {
	Command           string `json:"cmd"`
	URL               string `json:"url"`
	MaxTimeout        int64  `json:"maxTimeout"`
	ReturnOnlyCookies bool   `json:"returnOnlyCookies"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Cookies   []clearanceCookie `json:"cookies"`
		UserAgent string            `json:"userAgent"`
	} `json:"solution"`
}

type clearanceCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type clearance struct {
	Cookies   []clearanceCookie
	UserAgent string
}

type APIClient struct {
	config             Config
	log                *zap.Logger
	httpClient         *http.Client
	flareClient        *http.Client
	stateMu            sync.RWMutex
	state              session.State
	clearanceMu        sync.RWMutex
	clearance          *clearance
	clearanceRefreshMu sync.Mutex
}

func NewAPIClient(config Config, log *zap.Logger) *APIClient {
	return &APIClient{
		config: config,
		log:    log,
		state:  session.StatePending,
		httpClient: &http.Client{
			Timeout: config.APITimeout,
		},
		flareClient: &http.Client{Timeout: config.FlareSolverrTimeout + 5*time.Second},
	}
}

func (c *APIClient) FetchMap(ctx context.Context) ([]byte, error) {
	return c.fetch(ctx, "/map/icons", "map icons")
}

func (c *APIClient) FetchBusinesses(ctx context.Context) ([]byte, error) {
	return c.fetch(ctx, "/businesses/available", "available businesses")
}

func (c *APIClient) fetch(ctx context.Context, path, label string) ([]byte, error) {
	if c.config.APIToken == "" {
		return nil, c.fail(ErrAPITokenMissing)
	}

	endpoint := strings.TrimRight(c.config.APIBaseURL, "/") + path
	activeClearance := c.currentClearance()
	body, err := c.fetchEndpoint(ctx, endpoint, activeClearance)
	if err == nil {
		return c.succeed(body, label)
	}
	if !errors.Is(err, ErrAPICloudflareChallenge) || c.config.FlareSolverrURL == "" {
		return nil, c.fail(err)
	}

	c.clearanceRefreshMu.Lock()
	defer c.clearanceRefreshMu.Unlock()
	latestClearance := c.currentClearance()
	if !clearanceEqual(activeClearance, latestClearance) {
		body, retryErr := c.fetchEndpoint(ctx, endpoint, latestClearance)
		if retryErr == nil {
			return c.succeed(body, label)
		}
	}
	activeClearance, solveErr := c.solveCloudflare(ctx, endpoint)
	if solveErr != nil {
		return nil, c.fail(errors.Join(err, solveErr))
	}
	c.storeClearance(activeClearance)
	body, err = c.fetchEndpoint(ctx, endpoint, activeClearance)
	if err != nil {
		return nil, c.fail(fmt.Errorf("retry SARP %s with FlareSolverr clearance: %w", label, err))
	}
	return c.succeed(body, label)
}

func (c *APIClient) fetchEndpoint(ctx context.Context, endpoint string, activeClearance *clearance) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create SARP API request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")
	userAgent := c.config.APIUserAgent
	if activeClearance != nil {
		userAgent = activeClearance.UserAgent
		for _, cookie := range activeClearance.Cookies {
			request.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
		}
	}
	request.Header.Set("User-Agent", userAgent)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request SARP API: %w", err)
	}
	defer response.Body.Close()

	body, err := readLimited(response.Body, maxAPIResponseSize)
	if err != nil {
		return nil, fmt.Errorf("read SARP API response: %w", err)
	}
	if strings.EqualFold(response.Header.Get("cf-mitigated"), "challenge") {
		return nil, ErrAPICloudflareChallenge
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("SARP API returned HTTP %d", response.StatusCode)
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("SARP API returned a non-JSON response")
	}
	return body, nil
}

func (c *APIClient) solveCloudflare(ctx context.Context, endpoint string) (*clearance, error) {
	payload, err := json.Marshal(flareSolverrRequest{
		Command: "request.get", URL: endpoint,
		MaxTimeout: c.config.FlareSolverrTimeout.Milliseconds(), ReturnOnlyCookies: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode request: %v", ErrFlareSolverrFailed, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.FlareSolverrURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %v", ErrFlareSolverrFailed, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.flareClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: request service: %v", ErrFlareSolverrFailed, err)
	}
	defer response.Body.Close()
	body, err := readLimited(response.Body, maxFlareSolverrResponseSize)
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %v", ErrFlareSolverrFailed, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: service returned HTTP %d", ErrFlareSolverrFailed, response.StatusCode)
	}
	var result flareSolverrResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("%w: decode response: %v", ErrFlareSolverrFailed, err)
	}
	if result.Status != "ok" {
		return nil, fmt.Errorf("%w: %s", ErrFlareSolverrFailed, result.Message)
	}
	if result.Solution.UserAgent == "" || len(result.Solution.Cookies) == 0 {
		return nil, fmt.Errorf("%w: response omitted cookies or User-Agent", ErrFlareSolverrFailed)
	}
	validCookies := make([]clearanceCookie, 0, len(result.Solution.Cookies))
	for _, cookie := range result.Solution.Cookies {
		if cookie.Name != "" && cookie.Value != "" {
			validCookies = append(validCookies, cookie)
		}
	}
	if len(validCookies) == 0 {
		return nil, fmt.Errorf("%w: response contained no usable cookies", ErrFlareSolverrFailed)
	}
	c.log.Info("Cloudflare clearance refreshed through FlareSolverr", zap.Int("cookies", len(validCookies)))
	return &clearance{Cookies: validCookies, UserAgent: result.Solution.UserAgent}, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}

func (c *APIClient) succeed(body []byte, label string) ([]byte, error) {
	c.setState(session.StateAvailable)
	c.log.Debug("SARP resource fetched from API", zap.String("resource", label), zap.Int("bytes", len(body)))
	return body, nil
}

func (c *APIClient) currentClearance() *clearance {
	c.clearanceMu.RLock()
	defer c.clearanceMu.RUnlock()
	if c.clearance == nil {
		return nil
	}
	copy := *c.clearance
	copy.Cookies = append([]clearanceCookie(nil), c.clearance.Cookies...)
	return &copy
}

func (c *APIClient) storeClearance(value *clearance) {
	c.clearanceMu.Lock()
	c.clearance = value
	c.clearanceMu.Unlock()
}

func clearanceEqual(left, right *clearance) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.UserAgent != right.UserAgent || len(left.Cookies) != len(right.Cookies) {
		return false
	}
	for index := range left.Cookies {
		if left.Cookies[index] != right.Cookies[index] {
			return false
		}
	}
	return true
}

func (c *APIClient) Status() session.State {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *APIClient) fail(err error) error {
	c.setState(session.StatePending)
	return err
}

func (c *APIClient) setState(state session.State) {
	c.stateMu.Lock()
	c.state = state
	c.stateMu.Unlock()
}
