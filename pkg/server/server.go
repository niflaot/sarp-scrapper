package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	fiberzap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"github.com/oaswrap/spec/adapter/fiberv3openapi"
	"github.com/oaswrap/spec/option"
	uberzap "go.uber.org/zap"
)

const openAPIPath = "/openapi.json"
const docsPath = "/docs"

type Server struct {
	app       *fiber.App
	router    fiberv3openapi.Generator
	config    Config
	log       *uberzap.Logger
	listen    func(network, address string) (net.Listener, error)
	shutdown  func(context.Context) error
	serveDone chan struct{}
}

func New(config Config, log *uberzap.Logger) *Server {
	app := fiber.New(fiber.Config{AppName: "SARP Scrapper API"})
	app.Use(fiberzap.New(fiberzap.Config{Logger: log, Fields: []string{"ip", "latency", "status", "method", "url"}}))
	router := fiberv3openapi.NewRouter(app,
		option.WithTitle("SARP Scrapper API"),
		option.WithDescription("API for cached SARP houses available for sale."),
		option.WithVersion("1.0.0"),
		option.WithDocsPath(docsPath), option.WithSpecPath(openAPIPath), option.WithScalar(),
		option.WithDisableDocs(!config.IsDevelopment()),
	)
	return &Server{app: app, router: router, config: config, log: log, listen: net.Listen, shutdown: app.ShutdownWithContext, serveDone: make(chan struct{})}
}

func (s *Server) App() *fiber.App               { return s.app }
func (s *Server) Router() fiberv3openapi.Router { return s.router }

func (s *Server) start() error {
	if err := s.router.Validate(); err != nil {
		return fmt.Errorf("validate OpenAPI document: %w", err)
	}
	listener, err := s.listen("tcp", s.config.Address())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.config.Address(), err)
	}
	go func() {
		defer close(s.serveDone)
		if err := s.app.Listener(listener, fiber.ListenConfig{DisableStartupMessage: !s.config.IsDevelopment()}); err != nil {
			s.log.Error("Fiber server stopped unexpectedly", uberzap.Error(err))
		}
	}()
	s.log.Debug("SARP Scrapper API listening", uberzap.String("host", s.config.BindHost), uberzap.Uint16("port", s.config.BindPort), uberzap.String("environment", string(s.config.Environment)))
	return nil
}

func (s *Server) stop(ctx context.Context) error {
	err := s.shutdown(ctx)
	if errors.Is(err, fiber.ErrNotRunning) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("shutdown Fiber server: %w", err)
	}
	select {
	case <-s.serveDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for Fiber server shutdown: %w", ctx.Err())
	}
}
