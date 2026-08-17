package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestDocumentationEnvironment(t *testing.T) {
	for _, test := range []struct {
		environment Environment
		status      int
	}{{EnvironmentDevelopment, http.StatusOK}, {EnvironmentProduction, http.StatusNotFound}} {
		httpServer := New(Config{Environment: test.environment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
		httpServer.Router().Get("/test", func(ctx fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) }).With(option.OperationID("test"), option.Response(fiber.StatusNoContent, nil))
		request, _ := http.NewRequest(http.MethodGet, "/docs", nil)
		response, err := httpServer.App().Test(request)
		if err != nil {
			t.Fatalf("GET /docs: %v", err)
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != test.status {
			t.Fatalf("status = %d", response.StatusCode)
		}
		if test.environment == EnvironmentDevelopment && !strings.Contains(strings.ToLower(string(body)), "scalar") {
			t.Fatalf("unexpected docs: %q", body)
		}
	}
}

func TestStartupMessageUsesDebugLevel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := "http://" + listener.Addr().String() + "/test"
	core, observed := observer.New(zap.DebugLevel)
	httpServer := New(Config{Environment: EnvironmentProduction, BindHost: "127.0.0.1", BindPort: 8080}, zap.New(core))
	httpServer.Router().Get("/test", func(ctx fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) }).With(option.OperationID("startup-test"), option.Response(fiber.StatusNoContent, nil))
	httpServer.listen = func(string, string) (net.Listener, error) { return listener, nil }
	if err := httpServer.start(); err != nil {
		t.Fatalf("start() error = %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, requestErr := http.Get(address)
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusNoContent {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become ready: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := httpServer.stop(context.Background()); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
	entries := observed.FilterMessage("SARP Scrapper API listening").All()
	if len(entries) != 1 || entries[0].Level != zap.DebugLevel {
		t.Fatalf("unexpected startup logs: %#v", entries)
	}
}

func TestConfig(t *testing.T) {
	config := Config{Environment: EnvironmentDevelopment, BindHost: "::1", BindPort: 8080}
	if err := config.Validate(); err != nil || config.Address() != "[::1]:8080" || !config.IsDevelopment() {
		t.Fatalf("unexpected config: %#v, %v", config, err)
	}
	config.BindPort = 0
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() expected error")
	}
}
