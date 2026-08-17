package auth

import "github.com/niflaot/sarp-scrapper/pkg/session"

type LoginRequest struct {
	TwoFactorCode string `json:"twoFactorCode,omitempty" example:"123456" description:"Optional current 2FA code."`
}

type TwoFactorRequest struct {
	Code string `json:"code" required:"true" example:"123456" description:"Current 2FA code; never persisted."`
}

type Response struct {
	Status session.State `json:"status" example:"pending"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"login failed"`
}
