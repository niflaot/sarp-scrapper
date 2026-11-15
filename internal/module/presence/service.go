package presence

import (
	"context"
	"strconv"
	"strings"

	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/presence"
	"go.uber.org/zap"
)

type OnlineSource interface {
	Online(context.Context) ([]Character, error)
	OnlineAccounts(context.Context) ([]Account, error)
}

type Notifier interface {
	PresenceEnabled() bool
	NotifyPresence(context.Context, discord.PresenceNotice) error
	NotifyAccountPresence(context.Context, discord.AccountNotice) error
}

// Service notifies when a watched character goes from offline to online.
type Service struct{ tracker tracker[Character] }

func NewService(config presence.Config, source OnlineSource, notifier Notifier, log *zap.Logger) *Service {
	watched := make(map[string]struct{})
	for _, name := range config.Names() {
		watched[normalize(name)] = struct{}{}
	}
	recipient := config.DiscordUserID
	return &Service{tracker: tracker[Character]{
		kind:  "character",
		fetch: source.Online,
		key: func(character Character) (string, bool) {
			key := normalize(character.Name)
			_, ok := watched[key]
			return key, ok
		},
		notify: func(ctx context.Context, character Character) error {
			return notifier.NotifyPresence(ctx, discord.PresenceNotice{
				RecipientID: recipient, CharacterID: character.ID, Character: character.Name,
				Level: character.Level, Account: character.AccountName,
			})
		},
		log: log,
	}}
}

// Check compares the current online list with the previous one.
func (s *Service) Check(ctx context.Context) error { return s.tracker.check(ctx) }

// AccountService notifies when a watched SARP account shows up as online.
type AccountService struct{ tracker tracker[Account] }

func NewAccountService(config presence.Config, source OnlineSource, notifier Notifier, log *zap.Logger) *AccountService {
	watched := make(map[int64]struct{})
	for _, id := range config.AccountIDs() {
		watched[id] = struct{}{}
	}
	recipient := config.DiscordUserID
	return &AccountService{tracker: tracker[Account]{
		kind:  "account",
		fetch: source.OnlineAccounts,
		key: func(account Account) (string, bool) {
			_, ok := watched[account.ID]
			return strconv.FormatInt(account.ID, 10), ok
		},
		notify: func(ctx context.Context, account Account) error {
			return notifier.NotifyAccountPresence(ctx, discord.AccountNotice{
				RecipientID: recipient, AccountID: account.ID, AccountName: account.Name, AdminLevel: account.AdminLevel,
			})
		},
		log: log,
	}}
}

// Check compares the current online-users list with the previous one.
func (s *AccountService) Check(ctx context.Context) error { return s.tracker.check(ctx) }

func normalize(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
