package review

import "fmt"

type Config struct {
	StateFile string `env:"LISTINGS_STATE_FILE" envDefault:".cache/listings-state.json"`
}

func (c Config) Validate() error {
	if c.StateFile == "" {
		return fmt.Errorf("LISTINGS_STATE_FILE must not be empty")
	}
	return nil
}
