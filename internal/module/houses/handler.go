package houses

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/oaswrap/spec/option"
)

type ErrorResponse struct {
	Error string `json:"error" example:"house refresh failed"`
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func Register(server *server.Server, handler *Handler) {
	server.Router().Get("/houses", handler.list).With(
		option.OperationID("list-available-houses"),
		option.Summary("List available SARP houses"),
		option.Description("Returns the latest successful cached scrape and scheduler metadata."),
		option.Tags("houses"),
		option.Response(fiber.StatusOK, new(Snapshot)),
	)
	server.Router().Post("/houses/refresh", handler.refresh).With(
		option.OperationID("refresh-available-houses"),
		option.Summary("Refresh available SARP houses"),
		option.Tags("houses"),
		option.Response(fiber.StatusOK, new(Snapshot)),
		option.Response(fiber.StatusConflict, new(ErrorResponse)),
		option.Response(fiber.StatusServiceUnavailable, new(ErrorResponse)),
		option.Response(fiber.StatusBadGateway, new(ErrorResponse)),
	)
}

func (h *Handler) list(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(h.service.Snapshot())
}

func (h *Handler) refresh(ctx fiber.Ctx) error {
	err := h.service.Refresh(ctx.Context())
	if err == nil {
		return ctx.Status(fiber.StatusOK).JSON(h.service.Snapshot())
	}
	status := fiber.StatusBadGateway
	if errors.Is(err, ErrRefreshInProgress) {
		status = fiber.StatusConflict
	}
	if errors.Is(err, sarp.ErrDisabled) || errors.Is(err, sarp.ErrSessionPending) ||
		errors.Is(err, sarp.ErrAPITokenMissing) || errors.Is(err, sarp.ErrAPICloudflareChallenge) {
		status = fiber.StatusServiceUnavailable
	}
	return ctx.Status(status).JSON(ErrorResponse{Error: err.Error()})
}
