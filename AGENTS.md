# AGENTS.md

This file provides guidance to LLM agents when working with code in this repository.

## Commands

```bash
# Build
make build                    # builds bin/server
make generate                 # runs templ + tailwind (required before build if .templ files changed)

# Run (with .env)
make watch                    # hot-reload dev server via modd
make CMD='bin/server' run-with-env

# Test
go test ./...
go test ./internal/adapter/memory/...  # single package

# Release
make goreleaser               # snapshot release
```

**Config env prefix:** `XOLO_` (e.g. `XOLO_HTTP_ADDRESS=:3002`, `XOLO_STORAGE_DATABASE_DSN=data.sqlite`)

## Architecture

Xolo is an enterprise LLM gateway. It wraps the `github.com/bornholm/genai/proxy` OpenAI-compatible proxy with authentication, user management, and a web admin UI.

### Hexagonal structure

```
internal/
  core/
    model/      — domain types: User, AuthToken, Task
    port/       — interfaces: UserStore, TaskRunner, error sentinels
  adapter/
    gorm/       — SQLite/GORM implementation of UserStore
    cache/      — LRU cache wrappers for UserStore
    memory/     — in-memory TaskRunner implementation
  http/
    server.go   — HTTP server (CORS, slog middleware, context injection)
    options.go  — mount points configuration
    context/    — request-scoped values (BaseURL, CurrentURL, User)
    middleware/
      authn/    — authentication: OIDC (goth) + token-based
      authz/    — role assertions (user/admin/active)
      bridge/   — populates/creates User from authn identity
      ratelimit/— per-IP rate limiting
    handler/
      metrics/  — Prometheus metrics endpoint
      webui/
        common/ — shared assets handler + error pages + templ layout components
        admin/  — user management pages (admin-only)
        profile/— user profile + API token management
        templui/ — shadcn-style UI component library (templ)
  setup/        — wires config → concrete implementations (createFromConfigOnce pattern)
  config/       — env-based config structs (XOLO_ prefix)
  metrics/      — Prometheus metric definitions (namespace: "xolo")
```

### Key wiring

- `internal/setup/http_server.go` — assembles the full HTTP server from config; this is the composition root
- `internal/setup/helper.go` — `createFromConfigOnce` pattern: each dependency is created at most once per config
- The genai proxy is mounted at `/v1/` and sits behind auth middleware

### UI / templating

- Uses [`templ`](https://templ.guide/) for HTML templates (`*.templ` → `*_templ.go`)
- UI components are in `internal/http/handler/webui/templui/` (shadcn-inspired, via templui)
- Tailwind CSS is generated from `misc/tailwind/templui.css` → `internal/http/handler/webui/common/assets/templui.css`
- Run `make generate` after editing any `.templ` file or CSS

**IMPORTANT — composants UI :** toute nouvelle interface doit **obligatoirement** utiliser les composants templui disponibles sous `internal/http/handler/webui/templui/component/` (input, button, checkbox, label, card, badge, alert, etc.). Ne jamais utiliser de balises HTML brutes (`<input>`, `<button>`, `<select>`) là où un composant templui équivalent existe.

### Adding a new proxy hook

Mount hooks on the `proxy.Server` in `internal/setup/http_server.go` using `proxy.WithHook(...)`. The hook system is defined in `github.com/bornholm/genai/proxy` — see `proxy/hook.go` for the interface.
