package presence

import (
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{WatchNames: []string{" Harper_Brooks ", ""}, PollInterval: time.Minute, DiscordUserID: "278642935426449418"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if names := valid.Names(); len(names) != 1 || names[0] != "Harper_Brooks" {
		t.Fatalf("Names() = %#v", names)
	}

	disabled := Config{PollInterval: time.Minute}
	if disabled.Enabled() || disabled.Validate() != nil {
		t.Fatalf("empty watch list must be a valid disabled config")
	}

	for name, config := range map[string]Config{
		"missing user":  {WatchNames: []string{"A_B"}, PollInterval: time.Minute},
		"invalid user":  {WatchNames: []string{"A_B"}, PollInterval: time.Minute, DiscordUserID: "abc"},
		"short polling": {WatchNames: []string{"A_B"}, PollInterval: time.Second, DiscordUserID: "278642935426449418"},
	} {
		if err := config.Validate(); err == nil {
			t.Fatalf("%s: Validate() expected an error", name)
		}
	}
}
