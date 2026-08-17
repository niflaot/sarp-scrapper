// Package logger builds the service's structured Zap logger.
package logger

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

type Config struct {
	JSON  bool   `env:"LOG_JSON" envDefault:"false"`
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func (c Config) ParseLevel() (zapcore.Level, error) {
	var level zapcore.Level
	if err := level.Set(c.Level); err != nil {
		return level, fmt.Errorf("invalid LOG_LEVEL %q: %w", c.Level, err)
	}
	return level, nil
}

func (c Config) Validate() error {
	_, err := c.ParseLevel()
	return err
}
