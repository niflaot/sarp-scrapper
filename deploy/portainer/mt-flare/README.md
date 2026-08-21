# mt-flare

Portainer Stack for the FlareSolverr instance exposed by Caddy as
`https://flare.niflaot.dev`.

The Stack intentionally publishes no host port. Caddy must be attached to the
external `fl-network`. Once `mt-sarp-scrapper` is deployed, route its HTTP UI
and API paths separately while leaving FlareSolverr as the fallback:

```caddyfile
flare.niflaot.dev {
    @scraper path /panel* /listings* /houses* /businesses* /warehouses* /status
    handle @scraper {
        reverse_proxy mt-sarp-scrapper:3000
    }
    handle {
        reverse_proxy mt-flare:3000
    }
}
```

Copy `.env.example` into the Portainer Stack environment and deploy this folder.
Do not expose FlareSolverr without access controls to untrusted clients.
