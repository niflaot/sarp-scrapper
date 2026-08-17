package auth

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"github.com/oaswrap/spec/option"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func Register(server *server.Server, handler *Handler) {
	server.Router().Post("/auth/login", handler.login).With(
		option.OperationID("login-sarp"), option.Summary("Start or resume the SARP login"), option.Tags("auth"),
		option.Request(new(LoginRequest),
			option.ContentDescription("Optional 2FA code. Send an empty object to start with email and password only."),
			option.ContentExample(LoginRequest{TwoFactorCode: "123456"}),
		),
		option.Response(fiber.StatusOK, new(Response)), option.Response(fiber.StatusAccepted, new(Response)),
		option.Response(fiber.StatusBadRequest, new(ErrorResponse)), option.Response(fiber.StatusBadGateway, new(ErrorResponse)),
	)
	server.Router().Post("/auth/2fa", handler.twoFactor).With(
		option.OperationID("submit-sarp-2fa"), option.Summary("Submit a pending SARP 2FA code"), option.Tags("auth"),
		option.Request(new(TwoFactorRequest),
			option.ContentRequired(),
			option.ContentDescription("Current 2FA code for the browser session waiting in pending state."),
			option.ContentExample(TwoFactorRequest{Code: "123456"}),
		),
		option.Response(fiber.StatusOK, new(Response)), option.Response(fiber.StatusBadRequest, new(ErrorResponse)),
		option.Response(fiber.StatusConflict, new(ErrorResponse)), option.Response(fiber.StatusBadGateway, new(ErrorResponse)),
	)
}

func (h *Handler) login(ctx fiber.Ctx) error {
	var request LoginRequest
	if len(ctx.Body()) > 0 {
		if err := ctx.Bind().Body(&request); err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid JSON body"})
		}
	}
	state, err := h.service.Login(ctx.Context(), request.TwoFactorCode)
	if err != nil {
		return authError(ctx, err)
	}
	status := fiber.StatusOK
	if state == session.StatePending {
		status = fiber.StatusAccepted
	}
	return ctx.Status(status).JSON(Response{Status: state})
}

func (h *Handler) twoFactor(ctx fiber.Ctx) error {
	var request TwoFactorRequest
	if err := ctx.Bind().Body(&request); err != nil || request.Code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "code is required"})
	}
	state, err := h.service.SubmitTwoFactor(ctx.Context(), request.Code)
	if err != nil {
		return authError(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(Response{Status: state})
}

func authError(ctx fiber.Ctx, err error) error {
	status := fiber.StatusBadGateway
	if errors.Is(err, sarp.ErrDisabled) {
		status = fiber.StatusServiceUnavailable
	}
	if errors.Is(err, sarp.ErrNoPendingTwoFactor) {
		status = fiber.StatusConflict
	}
	if errors.Is(err, sarp.ErrInvalidTwoFactorCode) {
		status = fiber.StatusBadRequest
	}
	return ctx.Status(status).JSON(ErrorResponse{Error: err.Error()})
}
