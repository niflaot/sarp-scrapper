// Package status exposes the service availability endpoint.
package status

import (
	"github.com/gofiber/fiber/v3"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"github.com/oaswrap/spec/option"
)

type Response struct {
	Status session.State `json:"status" example:"available" description:"SARP session readiness."`
}

type Handler struct{ session session.Provider }

func Register(server *server.Server, provider session.Provider) {
	handler := &Handler{session: provider}
	server.Router().Get("/status", handler.get).With(
		option.OperationID("get-status"),
		option.Summary("Get service status"),
		option.Tags("status"),
		option.Response(fiber.StatusOK, new(Response)),
	)
}

func (h *Handler) get(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(Response{Status: h.session.Status()})
}
