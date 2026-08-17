# mt-flare

Portainer Stack for the FlareSolverr instance exposed by Caddy as
`https://flare.niflaot.dev`.

The Stack intentionally publishes no host port. Caddy must be attached to the
external `fl-network` and use this upstream:

```caddyfile
flare.niflaot.dev {
    reverse_proxy mt-flare:3000
}
```

Copy `.env.example` into the Portainer Stack environment and deploy this folder.
Do not expose FlareSolverr without access controls to untrusted clients.
