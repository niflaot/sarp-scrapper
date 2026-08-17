package auth

import (
	"context"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/session"
)

type Authenticator interface {
	Login(context.Context, string) (session.State, error)
	SubmitTwoFactor(context.Context, string) (session.State, error)
}

type Service struct{ authenticator Authenticator }

func NewService(browser *sarp.Browser) *Service { return newService(browser) }

func newService(authenticator Authenticator) *Service { return &Service{authenticator: authenticator} }

func (s *Service) Login(ctx context.Context, code string) (session.State, error) {
	return s.authenticator.Login(ctx, code)
}

func (s *Service) SubmitTwoFactor(ctx context.Context, code string) (session.State, error) {
	return s.authenticator.SubmitTwoFactor(ctx, code)
}
