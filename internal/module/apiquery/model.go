package apiquery

type Request struct {
	Provider string              `json:"provider,omitempty" example:"sarp" enum:"sarp,gta-rol" description:"Fixed API origin: sarp (default) or gta-rol."`
	Path     string              `json:"path" required:"true" example:"/account/profile" description:"Relative GET path below SARP_API_BASE_URL."`
	Query    map[string][]string `json:"query,omitempty" description:"Optional query parameters; each key may contain multiple values."`
}

type ErrorResponse struct {
	Error string `json:"error" example:"SARP API query failed"`
}
