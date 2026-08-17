package sarp

import "github.com/niflaot/sarp-scrapper/pkg/session"

type StatusProvider struct {
	config  Config
	api     *APIClient
	browser *Browser
}

func NewStatusProvider(config Config, api *APIClient, browser *Browser) *StatusProvider {
	return &StatusProvider{config: config, api: api, browser: browser}
}

func (p *StatusProvider) Status() session.State {
	if p.config.APIToken != "" && p.api.Status() == session.StateAvailable {
		return session.StateAvailable
	}
	return p.browser.Status()
}
