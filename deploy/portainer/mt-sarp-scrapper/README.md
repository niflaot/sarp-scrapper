# mt-sarp-scrapper

Portainer Stack that resolves `SARP_GIT_REF` as a remote branch, tag, or full
commit SHA, builds the Go binary only when the resolved commit changes, and runs
it on internal port `3000`.

This Stack deploys both `mt-flare` and `mt-sarp-scrapper`. They share the same
Docker network and host public egress IP, allowing the scraper to reuse the
Cloudflare clearance obtained internally from `http://mt-flare:3000/v1`.

Copy `.env.example` into Portainer, replace `API_TOKEN`, and preferably pin
`SARP_GIT_REF` to the delivered commit SHA. The service is available to other
containers on `fl-network` as `http://mt-sarp-scrapper:3000`.

The named volume `mt-sarp-scrapper-data` stores review decisions in
`/data/listings-state.json`. Removing the container or redeploying the Stack does
not remove ignored/reviewed state. Do not delete the volume unless you explicitly
want to reset all decisions and notification history.

`LISTINGS_HOUSE_MIN_VALUE`, `LISTINGS_BUSINESS_MIN_VALUE`, and
`LISTINGS_WAREHOUSE_MIN_VALUE` are inclusive
minimums. Results below them are omitted from the APIs, panel, cache reconciliation,
and Discord notifications. Their defaults are `500000`, `100000`, and `100000`
respectively.

Expose the panel through the existing Flare domain while keeping `/v1` and every
non-scraper path routed to FlareSolverr:

```caddyfile
flare.niflaot.dev {
    @scraper path /panel /panel/* /listings /listings/* /houses /houses/* /businesses /businesses/* /warehouses /warehouses/* /status
    handle @scraper {
        reverse_proxy mt-sarp-scrapper:3000
    }
    handle {
        reverse_proxy mt-flare:3000
    }
}
```
