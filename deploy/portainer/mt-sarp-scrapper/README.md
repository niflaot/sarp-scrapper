# mt-sarp-scrapper

Portainer Stack that resolves `SARP_GIT_REF` as a remote branch, tag, or full
commit SHA, builds the Go binary only when the resolved commit changes, and runs
it on internal port `3000`.

Deploy `mt-flare` first. Both containers must share the same host public egress
IP so the scraper can reuse the Cloudflare clearance obtained by
`https://flare.niflaot.dev/v1`.

Copy `.env.example` into Portainer, replace `API_TOKEN`, and preferably pin
`SARP_GIT_REF` to the delivered commit SHA. The service is available to other
containers on `fl-network` as `http://mt-sarp-scrapper:3000`.

The named volume `mt-sarp-scrapper-data` stores review decisions in
`/data/listings-state.json`. Removing the container or redeploying the Stack does
not remove ignored/reviewed state. Do not delete the volume unless you explicitly
want to reset all decisions and notification history.

To expose the panel through the existing Flare domain while keeping `/v1` routed
to FlareSolverr, use path-based Caddy routing:

```caddyfile
flare.niflaot.dev {
    @scraper path /panel* /listings* /houses* /businesses* /status
    handle @scraper {
        reverse_proxy mt-sarp-scrapper:3000
    }
    handle {
        reverse_proxy mt-flare:3000
    }
}
```
