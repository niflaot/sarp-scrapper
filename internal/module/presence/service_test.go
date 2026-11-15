package presence

import (
	"context"
	"errors"
	"testing"

	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/presence"
	"go.uber.org/zap"
)

type sourceStub struct {
	lists [][]Character
	err   error
	calls int
}

func (s *sourceStub) OnlineAccounts(context.Context) ([]Account, error) { return nil, nil }

func (s *sourceStub) Online(context.Context) ([]Character, error) {
	if s.err != nil {
		return nil, s.err
	}
	list := s.lists[min(s.calls, len(s.lists)-1)]
	s.calls++
	return list, nil
}

type notifierStub struct {
	notices []discord.PresenceNotice
	err     error
}

func (n *notifierStub) PresenceEnabled() bool { return true }
func (n *notifierStub) NotifyAccountPresence(context.Context, discord.AccountNotice) error {
	return nil
}
func (n *notifierStub) NotifyPresence(_ context.Context, notice discord.PresenceNotice) error {
	if n.err != nil {
		return n.err
	}
	n.notices = append(n.notices, notice)
	return nil
}

func newTestService(source OnlineSource, notifier Notifier) *Service {
	config := presence.Config{WatchNames: []string{"Harper_Brooks", "orla_murphy"}, DiscordUserID: "278642935426449418"}
	return NewService(config, source, notifier, zap.NewNop())
}

func online(names ...string) []Character {
	characters := make([]Character, 0, len(names))
	for index, name := range names {
		characters = append(characters, Character{ID: int64(index + 1), Name: name, Level: 10, AccountName: "Acc"})
	}
	return characters
}

func TestCheckNotifiesOnlyOnConnection(t *testing.T) {
	source := &sourceStub{lists: [][]Character{
		online("Adam_Belfort", "Orla_Murphy"),                  // baseline: Orla already online
		online("Adam_Belfort", "Orla_Murphy", "Harper_Brooks"), // Harper connects
		online("Adam_Belfort", "Orla_Murphy", "Harper_Brooks"), // nothing new
		online("Adam_Belfort"),                                 // both leave
		online("ORLA_MURPHY"),                                  // Orla returns (case-insensitive)
	}}
	notifier := &notifierStub{}
	service := newTestService(source, notifier)

	wantNotices := []int{0, 1, 1, 1, 2}
	for step, want := range wantNotices {
		if err := service.Check(context.Background()); err != nil {
			t.Fatalf("step %d: Check() error = %v", step, err)
		}
		if len(notifier.notices) != want {
			t.Fatalf("step %d: notices = %d, want %d (%#v)", step, len(notifier.notices), want, notifier.notices)
		}
	}
	if notice := notifier.notices[0]; notice.Character != "Harper_Brooks" || notice.RecipientID != "278642935426449418" || notice.Account != "Acc" {
		t.Fatalf("unexpected notice: %#v", notice)
	}
}

func TestCheckRetriesFailedNotification(t *testing.T) {
	source := &sourceStub{lists: [][]Character{online(), online("Harper_Brooks"), online("Harper_Brooks")}}
	notifier := &notifierStub{}
	service := newTestService(source, notifier)

	if err := service.Check(context.Background()); err != nil {
		t.Fatalf("baseline error = %v", err)
	}
	notifier.err = errors.New("discord down")
	if err := service.Check(context.Background()); err == nil {
		t.Fatal("Check() expected the notification error")
	}
	notifier.err = nil
	if err := service.Check(context.Background()); err != nil || len(notifier.notices) != 1 {
		t.Fatalf("retry: err = %v, notices = %#v", err, notifier.notices)
	}
}

func TestCheckKeepsStateWhenQueryFails(t *testing.T) {
	source := &sourceStub{lists: [][]Character{online("Harper_Brooks")}}
	notifier := &notifierStub{}
	service := newTestService(source, notifier)
	if err := service.Check(context.Background()); err != nil {
		t.Fatalf("baseline error = %v", err)
	}
	source.err = errors.New("HTTP 503")
	if err := service.Check(context.Background()); err == nil {
		t.Fatal("Check() expected the query error")
	}
	source.err = nil
	if err := service.Check(context.Background()); err != nil || len(notifier.notices) != 0 {
		t.Fatalf("a failed query must not look like a reconnection: err = %v, notices = %#v", err, notifier.notices)
	}
}

type accountSourceStub struct {
	sourceStub
	lists [][]Account
	calls int
	err   error
}

func (s *accountSourceStub) OnlineAccounts(context.Context) ([]Account, error) {
	if s.err != nil {
		return nil, s.err
	}
	list := s.lists[min(s.calls, len(s.lists)-1)]
	s.calls++
	return list, nil
}

type accountNotifierStub struct {
	notifierStub
	accounts []discord.AccountNotice
	err      error
}

func (n *accountNotifierStub) NotifyAccountPresence(_ context.Context, notice discord.AccountNotice) error {
	if n.err != nil {
		return n.err
	}
	n.accounts = append(n.accounts, notice)
	return nil
}

func TestAccountServiceNotifiesWhenWatchedAccountAppears(t *testing.T) {
	account := func(id int64, name string) Account { return Account{ID: id, Name: name, AdminLevel: 1} }
	source := &accountSourceStub{lists: [][]Account{
		{account(1, "varky"), account(15316, "known")},                        // baseline: 15316 already listed
		{account(1, "varky"), account(15316, "known"), account(13297, "new")}, // 13297 appears
		{account(1, "varky"), account(13297, "new")},                          // nothing new
		{account(1, "varky")},                                                 // both leave
		{account(15316, "known")},                                             // 15316 returns
	}}
	notifier := &accountNotifierStub{}
	config := presence.Config{WatchAccountIDs: []string{"13297", "15316"}, DiscordUserID: "278642935426449418"}
	service := NewAccountService(config, source, notifier, zap.NewNop())

	for step, want := range []int{0, 1, 1, 1, 2} {
		if err := service.Check(context.Background()); err != nil {
			t.Fatalf("step %d: Check() error = %v", step, err)
		}
		if len(notifier.accounts) != want {
			t.Fatalf("step %d: notices = %d, want %d (%#v)", step, len(notifier.accounts), want, notifier.accounts)
		}
	}
	if got := notifier.accounts[0]; got.AccountID != 13297 || got.AccountName != "new" || got.RecipientID != "278642935426449418" || got.AdminLevel != 1 {
		t.Fatalf("unexpected notice: %#v", got)
	}
}

func TestAccountServiceKeepsStateOnFailures(t *testing.T) {
	source := &accountSourceStub{lists: [][]Account{{}, {{ID: 13297, Name: "x"}}, {{ID: 13297, Name: "x"}}}}
	notifier := &accountNotifierStub{}
	service := NewAccountService(presence.Config{WatchAccountIDs: []string{"13297"}, DiscordUserID: "278642935426449418"}, source, notifier, zap.NewNop())
	if err := service.Check(context.Background()); err != nil {
		t.Fatalf("baseline error = %v", err)
	}
	notifier.err = errors.New("discord down")
	if err := service.Check(context.Background()); err == nil {
		t.Fatal("Check() expected the notification error")
	}
	notifier.err = nil
	if err := service.Check(context.Background()); err != nil || len(notifier.accounts) != 1 {
		t.Fatalf("retry: err = %v, notices = %#v", err, notifier.accounts)
	}
	source.err = errors.New("HTTP 503")
	if err := service.Check(context.Background()); err == nil {
		t.Fatal("Check() expected the query error")
	}
}
