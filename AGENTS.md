# AGENTS.md

Guidance for humans and AI agents working in this repository.

## Purpose

Keep the service modular, easy to test, and independent of delivery mechanisms
or infrastructure details. Prefer small changes that extend an existing module
over broad shared abstractions.

## Project Layout

```text
cmd/                            Process entry point and root composition
internal/module/<domain>/       Domain-specific application modules
pkg/config/                     Composed application configuration loader
pkg/<component>/                Reusable infrastructure components
```

Domain modules own their handlers, services, repository contracts, and Fx
wiring. Cross-cutting infrastructure such as logging, HTTP, and remote clients
belongs under `pkg/`.

## Conventions

- Pass `context.Context` through I/O operations.
- Keep Fiber handlers thin and register routes through the OpenAPI router.
- Use Fx only for application composition and lifecycle management.
- Keep configuration modular; `.env` is loaded only by `pkg/config`.
- Never commit credentials, tokens, cookies, or captured production responses.
- Run `go fmt`, `go test -race`, and `go vet` before finishing.
