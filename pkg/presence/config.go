// Package presence holds the configuration of the character connection watcher.
package presence

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var snowflake = regexp.MustCompile(`^[0-9]{15,25}$`)

const minPollInterval = 10 * time.Second

type Config struct {
	WatchNames    []string      `env:"PRESENCE_WATCH_NAMES" envSeparator:","`
	PollInterval  time.Duration `env:"PRESENCE_POLL_INTERVAL" envDefault:"60s"`
	DiscordUserID string        `env:"PRESENCE_DISCORD_USER_ID"`
}

// Enabled reports whether at least one character is being watched.
func (c Config) Enabled() bool { return len(c.Names()) > 0 }

// Names returns the watched character names without blanks.
func (c Config) Names() []string {
	names := make([]string, 0, len(c.WatchNames))
	for _, name := range c.WatchNames {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (c Config) Validate() error {
	if c.PollInterval < minPollInterval {
		return fmt.Errorf("PRESENCE_POLL_INTERVAL must be at least %s", minPollInterval)
	}
	if !c.Enabled() {
		return nil
	}
	if !snowflake.MatchString(c.DiscordUserID) {
		return fmt.Errorf("PRESENCE_DISCORD_USER_ID must be a Discord user ID when PRESENCE_WATCH_NAMES is set")
	}
	return nil
}
