package listings

import (
	"bytes"
	"errors"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"github.com/oaswrap/spec/option"
)

type ListResponse struct {
	Listings []review.Record `json:"listings"`
	Count    int             `json:"count"`
}

type ActionErrorResponse struct {
	Error string `json:"error"`
}

type Handler struct {
	service  *Service
	template *template.Template
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, template: template.Must(template.New("panel").Funcs(template.FuncMap{
		"money": formatMoney, "iconURL": panelIconURL,
	}).Parse(panelHTML))}
}

func Register(server *server.Server, handler *Handler) {
	server.Router().Get("/listings", handler.list).With(
		option.OperationID("list-reviewed-listings"), option.Summary("List available reviewed listings"), option.Tags("listings"),
		option.Response(fiber.StatusOK, new(ListResponse)),
	)
	server.Router().Get("/panel", handler.panel).With(
		option.OperationID("get-listings-panel"), option.Summary("Open the listings review panel"), option.Tags("listings"),
		option.Response(fiber.StatusOK, new(string)),
	)
	server.Router().Post("/listings/:kind/:key/ignore", handler.ignore).With(
		option.OperationID("ignore-listing"), option.Summary("Permanently ignore an available listing"), option.Tags("listings"),
		option.Response(fiber.StatusOK, new(review.Record)), option.Response(fiber.StatusNotFound, new(ActionErrorResponse)),
	)
	server.Router().Post("/listings/:kind/:key/review", handler.review).With(
		option.OperationID("review-listing"), option.Summary("Mark an available listing as reviewed"), option.Tags("listings"),
		option.Response(fiber.StatusOK, new(review.Record)), option.Response(fiber.StatusNotFound, new(ActionErrorResponse)),
	)
}

func (h *Handler) list(ctx fiber.Ctx) error {
	records := h.service.Available()
	return ctx.Status(fiber.StatusOK).JSON(ListResponse{Listings: records, Count: len(records)})
}

func (h *Handler) panel(ctx fiber.Ctx) error {
	var output bytes.Buffer
	if err := h.template.Execute(&output, struct{ Listings []review.Record }{Listings: h.service.Available()}); err != nil {
		return err
	}
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return ctx.Status(fiber.StatusOK).Send(output.Bytes())
}

func (h *Handler) ignore(ctx fiber.Ctx) error { return h.setStatus(ctx, review.StatusIgnored) }
func (h *Handler) review(ctx fiber.Ctx) error { return h.setStatus(ctx, review.StatusReviewed) }

func (h *Handler) setStatus(ctx fiber.Ctx, status review.Status) error {
	kind, key := ctx.Params("kind"), ctx.Params("key")
	if (kind != KindHouse && kind != KindBusiness && kind != KindWarehouse) || key == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(ActionErrorResponse{Error: "kind and key are invalid"})
	}
	record, err := h.service.SetStatus(kind, key, status)
	if err != nil {
		code := fiber.StatusInternalServerError
		if errors.Is(err, review.ErrListingNotFound) {
			code = fiber.StatusNotFound
		}
		return ctx.Status(code).JSON(ActionErrorResponse{Error: err.Error()})
	}
	if strings.Contains(ctx.Get(fiber.HeaderAccept), fiber.MIMETextHTML) || strings.HasPrefix(ctx.Get(fiber.HeaderContentType), fiber.MIMEApplicationForm) {
		ctx.Set(fiber.HeaderLocation, "/panel")
		return ctx.SendStatus(fiber.StatusSeeOther)
	}
	return ctx.Status(fiber.StatusOK).JSON(record)
}

func formatMoney(value int64) string {
	digits := strconv.FormatInt(value, 10)
	start := 0
	if strings.HasPrefix(digits, "-") {
		start = 1
	}
	for index := len(digits) - 3; index > start; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return "$" + digits
}

func panelIconURL(value string) string {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}
	return value
}

const panelHTML = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Disponibles · SARP</title>
  <style>
    :root{color-scheme:light;--bg:#eef1f5;--card:#fff;--line:#dfe3e9;--text:#303030;--muted:#6f7785;--blue:#316be8;--red:#d94b59;--green:#20ad5b}
    *{box-sizing:border-box}body{margin:0;background:linear-gradient(180deg,#f7f8fa,var(--bg));color:var(--text);font:15px/1.5 system-ui,-apple-system,sans-serif}
    main{width:min(1120px,calc(100% - 32px));margin:48px auto}.top{display:flex;align-items:end;justify-content:space-between;gap:24px;margin-bottom:24px}
    h1{font-size:clamp(30px,5vw,52px);letter-spacing:-.04em;margin:0}.subtitle{color:var(--muted);margin:6px 0 0}.count{color:var(--muted);white-space:nowrap}
    .grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:18px}.card{background:var(--card);border:1px solid var(--line);border-radius:18px;padding:22px;box-shadow:0 12px 32px #26324d18}
    .head{display:flex;gap:14px;align-items:flex-start}.icon{width:54px;height:54px;border-radius:12px;object-fit:contain;background:#f5f7fa;border:1px solid var(--line)}.placeholder{display:grid;place-items:center;font-size:24px}
    .summary{min-width:0;flex:1}h2{font-size:21px;line-height:1.2;margin:0 0 2px;font-weight:800}.listing-name{font-size:20px;line-height:1.28;margin:0;color:#353535}.price{font-size:22px;line-height:1.2;margin:5px 0 12px;color:var(--green);font-weight:850}.badges{display:flex;gap:7px;flex-wrap:wrap}.badge{padding:3px 8px;border-radius:999px;background:#edf1f6;color:#596579;font-size:12px}.reviewed{background:#dcf7e9;color:#157746}
    .actions{display:flex;gap:9px;margin-top:18px;padding-top:15px;border-top:1px solid var(--line)}.actions form{flex:1}.button{width:100%;border:0;border-radius:11px;padding:10px 12px;color:white;font-weight:700;cursor:pointer}.review{background:var(--blue)}.ignore{background:transparent;color:var(--red);border:1px solid #e6a7ad}.button:disabled{opacity:.45;cursor:default}
    .empty{grid-column:1/-1;text-align:center;color:var(--muted);padding:70px 20px;border:1px dashed var(--line);border-radius:18px;background:white}
    @media(max-width:600px){main{margin:28px auto}.top{align-items:start;flex-direction:column;gap:4px}}
  </style>
</head>
<body><main>
  <div class="top"><div><h1>Disponibles</h1><p class="subtitle">Propiedades, negocios y almacenes encontrados por SARP.</p></div><div class="count">{{len .Listings}} resultados</div></div>
  <section class="grid">
  {{range .Listings}}
    <article class="card">
      <div class="head">
        {{with iconURL .Icon}}<img class="icon" src="{{.}}" alt="">{{else}}<div class="icon placeholder">{{if eq .Kind "business"}}🏪{{else if eq .Kind "warehouse"}}🏬{{else}}🏠{{end}}</div>{{end}}
        <div class="summary"><h2>{{if eq .Kind "business"}}Negocio {{.Key}}{{else if eq .Kind "warehouse"}}Almacén {{.Key}}{{else}}Llave {{.Key}}{{end}}</h2><p class="listing-name">{{.Name}}</p><p class="price">{{money .Value}}</p><div class="badges"><span class="badge">{{if eq .Kind "business"}}Negocio{{else if eq .Kind "warehouse"}}Almacén{{else}}Propiedad{{end}}</span><span class="badge {{if eq .Status "reviewed"}}reviewed{{end}}">{{if eq .Status "reviewed"}}Revisado{{else}}Nuevo{{end}}</span></div></div>
      </div>
      <div class="actions">
        <form method="post" action="/listings/{{.Kind}}/{{.Key}}/review"><button class="button review" {{if eq .Status "reviewed"}}disabled{{end}}>Revisar</button></form>
        <form method="post" action="/listings/{{.Kind}}/{{.Key}}/ignore"><button class="button ignore">Ignorar</button></form>
      </div>
    </article>
  {{else}}<div class="empty">No hay propiedades, negocios o almacenes disponibles.</div>{{end}}
  </section>
</main></body></html>`
