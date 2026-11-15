package presence

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{WatchNames: []string{" Harper_Brooks ", ""}, PollInterval: time.Minute, AccountPollInterval: 5 * time.Minute, DiscordUserID: "278642935426449418"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if names := valid.Names(); len(names) != 1 || names[0] != "Harper_Brooks" {
		t.Fatalf("Names() = %#v", names)
	}

	disabled := Config{PollInterval: time.Minute, AccountPollInterval: 5 * time.Minute}
	if disabled.Enabled() || disabled.Validate() != nil {
		t.Fatalf("empty watch list must be a valid disabled config")
	}

	accounts := Config{WatchAccountIDs: []string{" 13297", "15316 ", ""}, PollInterval: time.Minute, AccountPollInterval: 5 * time.Minute, DiscordUserID: "278642935426449418"}
	if err := accounts.Validate(); err != nil || !accounts.AccountsEnabled() {
		t.Fatalf("accounts config: enabled = %v, err = %v", accounts.AccountsEnabled(), err)
	}
	if ids := accounts.AccountIDs(); len(ids) != 2 || ids[0] != 13297 || ids[1] != 15316 {
		t.Fatalf("AccountIDs() = %#v", ids)
	}

	for name, config := range map[string]Config{
		"accounts without user": {WatchAccountIDs: []string{"13297"}, PollInterval: time.Minute, AccountPollInterval: time.Minute},
		"bad account id":        {WatchAccountIDs: []string{"abc"}, PollInterval: time.Minute, AccountPollInterval: time.Minute, DiscordUserID: "278642935426449418"},
		"short account polling": {WatchAccountIDs: []string{"1"}, PollInterval: time.Minute, AccountPollInterval: time.Second, DiscordUserID: "278642935426449418"},
		"missing user":          {WatchNames: []string{"A_B"}, PollInterval: time.Minute, AccountPollInterval: 5 * time.Minute},
		"invalid user":          {WatchNames: []string{"A_B"}, PollInterval: time.Minute, AccountPollInterval: 5 * time.Minute, DiscordUserID: "abc"},
		"short polling":         {WatchNames: []string{"A_B"}, PollInterval: time.Second, DiscordUserID: "278642935426449418"},
	} {
		if err := config.Validate(); err == nil {
			t.Fatalf("%s: Validate() expected an error", name)
		}
	}
}
