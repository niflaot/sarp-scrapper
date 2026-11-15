// Package presence holds the configuration of the connection watchers.
package presence

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var snowflake = regexp.MustCompile(`^[0-9]{15,25}$`)

const minPollInterval = 10 * time.Second

type Config struct {
	// Characters watched through the global online-players list.
	WatchNames   []string      `env:"PRESENCE_WATCH_NAMES" envSeparator:","`
	PollInterval time.Duration `env:"PRESENCE_POLL_INTERVAL" envDefault:"60s"`
	// SARP account IDs watched through the home online-users list.
	WatchAccountIDs     []string      `env:"PRESENCE_WATCH_ACCOUNT_IDS" envSeparator:","`
	AccountPollInterval time.Duration `env:"PRESENCE_ACCOUNT_POLL_INTERVAL" envDefault:"5m"`
	DiscordUserID       string        `env:"PRESENCE_DISCORD_USER_ID"`
}

// Enabled reports whether at least one character is being watched.
func (c Config) Enabled() bool { return len(c.Names()) > 0 }

// AccountsEnabled reports whether at least one account is being watched.
func (c Config) AccountsEnabled() bool { return len(c.WatchAccountIDs) > 0 && len(c.AccountIDs()) > 0 }

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

// AccountIDs returns the watched account IDs. Invalid entries are rejected by
// Validate and skipped here.
func (c Config) AccountIDs() []int64 {
	ids := make([]int64, 0, len(c.WatchAccountIDs))
	for _, raw := range c.WatchAccountIDs {
		if id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (c Config) Validate() error {
	if c.PollInterval < minPollInterval {
		return fmt.Errorf("PRESENCE_POLL_INTERVAL must be at least %s", minPollInterval)
	}
	if c.AccountPollInterval < minPollInterval {
		return fmt.Errorf("PRESENCE_ACCOUNT_POLL_INTERVAL must be at least %s", minPollInterval)
	}
	for _, raw := range c.WatchAccountIDs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if id, err := strconv.ParseInt(raw, 10, 64); err != nil || id <= 0 {
			return fmt.Errorf("PRESENCE_WATCH_ACCOUNT_IDS contains an invalid account ID %q", raw)
		}
	}
	if !c.Enabled() && !c.AccountsEnabled() {
		return nil
	}
	if !snowflake.MatchString(c.DiscordUserID) {
		return fmt.Errorf("PRESENCE_DISCORD_USER_ID must be a Discord user ID when PRESENCE_WATCH_NAMES or PRESENCE_WATCH_ACCOUNT_IDS is set")
	}
	return nil
}
