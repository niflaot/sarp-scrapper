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
