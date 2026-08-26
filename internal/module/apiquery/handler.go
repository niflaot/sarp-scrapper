package apiquery

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/oaswrap/spec/option"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func Register(server *server.Server, handler *Handler) {
	server.Router().Post("/api/query", handler.query).With(
		option.OperationID("query-sarp-api"),
		option.Summary("Run an authenticated GET against the configured SARP API"),
		option.Tags("api"),
		option.Request(new(Request),
			option.ContentRequired(),
			option.ContentDescription("Only relative paths below SARP_API_BASE_URL are accepted; absolute URLs and traversal are rejected."),
			option.ContentExample(Request{Path: "/character/search", Query: map[string][]string{"name": {"John"}}}),
		),
		option.Response(fiber.StatusOK, new(map[string]any)),
		option.Response(fiber.StatusBadRequest, new(ErrorResponse)),
		option.Response(fiber.StatusServiceUnavailable, new(ErrorResponse)),
		option.Response(fiber.StatusBadGateway, new(ErrorResponse)),
	)
}

func (h *Handler) query(ctx fiber.Ctx) error {
	var request Request
	if err := ctx.Bind().Body(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid JSON body"})
	}
	body, err := h.service.Query(ctx.Context(), request)
	if err != nil {
		status := fiber.StatusBadGateway
		if errors.Is(err, ErrInvalidRequest) || errors.Is(err, sarp.ErrInvalidAPIPath) {
			status = fiber.StatusBadRequest
		}
		if errors.Is(err, sarp.ErrAPITokenMissing) || errors.Is(err, sarp.ErrAPICloudflareChallenge) || errors.Is(err, sarp.ErrFlareSolverrFailed) {
			status = fiber.StatusServiceUnavailable
		}
		return ctx.Status(status).JSON(ErrorResponse{Error: err.Error()})
	}
	ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return ctx.Status(fiber.StatusOK).Send(body)
}
