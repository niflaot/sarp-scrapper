// Package config loads and composes configuration owned by service components.
package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/niflaot/sarp-scrapper/pkg/logger"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
)

type Config struct {
	Server server.Config
	Logger logger.Config
	SARP   sarp.Config
}

func Load() (*Config, error) { return load(".env") }

func load(path string) (*Config, error) {
	if err := godotenv.Load(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load environment file %q: %w", path, err)
	}
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("parse environment: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server configuration: %w", err)
	}
	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("logger configuration: %w", err)
	}
	if err := c.SARP.Validate(); err != nil {
		return fmt.Errorf("SARP configuration: %w", err)
	}
	return nil
}

func Server(c *Config) server.Config { return c.Server }
func Logger(c *Config) logger.Config { return c.Logger }
func SARP(c *Config) sarp.Config     { return c.SARP }
