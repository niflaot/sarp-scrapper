// Package server owns the HTTP transport and its network configuration.
package server

import (
	"fmt"
	"net"
	"strconv"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment Environment `env:"APP_ENVIRONMENT" envDefault:"development"`
	BindHost    string      `env:"BIND_HOST" envDefault:"0.0.0.0"`
	BindPort    uint16      `env:"BIND_PORT" envDefault:"8080"`
}

func (c Config) Address() string {
	return net.JoinHostPort(c.BindHost, strconv.FormatUint(uint64(c.BindPort), 10))
}
func (c Config) IsDevelopment() bool { return c.Environment == EnvironmentDevelopment }

func (c Config) Validate() error {
	switch c.Environment {
	case EnvironmentDevelopment, EnvironmentProduction:
	default:
		return fmt.Errorf("APP_ENVIRONMENT must be %q or %q, got %q", EnvironmentDevelopment, EnvironmentProduction, c.Environment)
	}
	if c.BindHost == "" {
		return fmt.Errorf("BIND_HOST must not be empty")
	}
	if c.BindPort == 0 {
		return fmt.Errorf("BIND_PORT must be greater than zero")
	}
	return nil
}
