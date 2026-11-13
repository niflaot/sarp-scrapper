package presence

import (
	"context"
	"errors"
	"strings"

	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/presence"
	"go.uber.org/zap"
)

type OnlineSource interface {
	Online(context.Context) ([]Character, error)
}

type Notifier interface {
	PresenceEnabled() bool
	NotifyPresence(context.Context, discord.PresenceNotice) error
}

// Service notifies when a watched character goes from offline to online. It
// keeps its state in memory and must be driven by a single goroutine.
type Service struct {
	source    OnlineSource
	notifier  Notifier
	log       *zap.Logger
	watched   map[string]struct{}
	recipient string

	online map[string]struct{}
	primed bool
}

func NewService(config presence.Config, source OnlineSource, notifier Notifier, log *zap.Logger) *Service {
	watched := make(map[string]struct{})
	for _, name := range config.Names() {
		watched[normalize(name)] = struct{}{}
	}
	return &Service{source: source, notifier: notifier, log: log, watched: watched, recipient: config.DiscordUserID}
}

// Check compares the current online list with the previous one. The first
// successful check only records who is already online. A failed query keeps the
// previous state, and a failed notification is retried on the next check.
func (s *Service) Check(ctx context.Context) error {
	characters, err := s.source.Online(ctx)
	if err != nil {
		return err
	}
	current := make(map[string]Character)
	for _, character := range characters {
		key := normalize(character.Name)
		if _, ok := s.watched[key]; ok {
			current[key] = character
		}
	}
	if !s.primed {
		s.online = make(map[string]struct{}, len(current))
		for key := range current {
			s.online[key] = struct{}{}
		}
		s.primed = true
		s.log.Info("presence baseline recorded", zap.Int("watched_online", len(current)))
		return nil
	}

	next := make(map[string]struct{}, len(current))
	var failures []error
	for key, character := range current {
		if _, was := s.online[key]; was {
			next[key] = struct{}{}
			continue
		}
		err := s.notifier.NotifyPresence(ctx, discord.PresenceNotice{
			RecipientID: s.recipient, CharacterID: character.ID, Character: character.Name,
			Level: character.Level, Account: character.AccountName,
		})
		if err != nil {
			s.log.Warn("presence notification failed; it will be retried", zap.String("character", character.Name), zap.Error(err))
			failures = append(failures, err)
			continue
		}
		s.log.Info("watched character connected", zap.String("character", character.Name))
		next[key] = struct{}{}
	}
	s.online = next
	return errors.Join(failures...)
}

func normalize(name string) string { return strings.ToLower(strings.TrimSpace(name)) }
