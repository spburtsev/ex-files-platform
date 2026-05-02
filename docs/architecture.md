# Architecture

## Project Overview

Ex-Files is a document notarization & approval platform. Users organize work into **workspaces / issues / documents**, with versioning, review workflows, and an append-only audit log. Three roles: root, manager, employee.

## Monorepo Structure

- **api/** OpenAPI 3.0.3 specification - single source of truth for all API contracts
- **ex-files-backend/** Go HTTP API (stdlib `net/http`, ogen-generated server, GORM, PostgreSQL, MinIO)
- **ex-files-frontend/** SvelteKit 5 app (Svelte 5, Tailwind CSS 4, TypeScript, bun)

## Spec-First Communication

All API operations and schemas are defined in `api/openapi.yaml`. Code generation produces server and client artefacts:

- `make openapi-gen-go` - runs [ogen](https://ogen.dev) to produce Go server skeleton, types, and request/response wiring at `ex-files-backend/oapi/`
- `make openapi-gen-ts` - runs [@hey-api/openapi-ts](https://heyapi.dev) to produce a typed fetch SDK at `ex-files-frontend/src/lib/api/`
- `make openapi-lint` - validates the spec with redocly
- `make openapi-docs` - renders human-readable HTML at `docs/api/index.html`

Backend handlers implement the ogen `Handler` interface and return typed response structs. Responses are serialised as JSON. Frontend remote functions (`queries.remote.ts`, `commands.remote.ts`) call the generated SDK rather than constructing fetches by hand.

The `/events` SSE endpoint is intentionally outside the spec (OpenAPI does not model `text/event-stream` cleanly) and is handled by a hand-written `http.Handler` mounted alongside the ogen server.

## Backend Layers

- **handlers/** - Implementations of the ogen `Handler` and `SecurityHandler` interfaces. The `Server` struct in `handlers/server.go` is the single root that ogen drives.
- **oapi/** - ogen-generated server, types, decoders/encoders. Gitignored; rebuilt by `make openapi-gen-go`.
- **services/** - Repository interfaces + GORM implementations, token service, bcrypt hasher, MinIO storage, SSE hub
- **models/** - GORM models with auto-migration on startup
- **middleware/** - Stdlib `http.Handler` middleware: JWT auth context, request logger, recovery, cookie jar (used to set/clear `session` on responses)
- **seed/** - Creates default users on startup

The router is a tiny `http.ServeMux` in `main.go`: `/healthz` and `/events` are mounted directly; everything else flows through the ogen handler. Cross-cutting concerns wrap the mux: CORS → recovery → request logger → cookie jar.

Services implement interfaces defined in `services/interfaces.go` to enable dependency injection and mocking.

## Frontend Patterns

- **Generated SDK**: `src/lib/api/` contains the typed fetch client (gitignored, rebuilt by `bun run gen:api`)
- **API runtime config**: `src/lib/api-client.ts` exports `createClientConfig` (base URL) and `apiOpts()` (per-call helper that threads SvelteKit `event.fetch` and forwards the session cookie as a Bearer token)
- **Remote functions**: `src/lib/queries.remote.ts` (queries via `query()`) and `src/lib/commands.remote.ts` (mutations via `command()`) - SvelteKit server functions that call the generated SDK
- **Format helpers**: `src/lib/utils.ts` - `formatTimestamp(iso)`, `isManager(role)`, `roleName(role)`, `initials(name)` (the file is named for historical reasons; the helpers operate on plain strings now)
- **i18n**: Paraglide JS with `en` and `pl` locales. Messages in `messages/{locale}.json`. Detection: URL → cookie → base locale
- **UI components**: shadcn-svelte (bits-ui) in `src/lib/components/ui/`
- **Auth**: Session cookie (`session`, HTTP-only, 8h TTL). Server hook in `hooks.server.ts` validates and redirects unauthenticated users
- **Routing**: `(auth)/` group for login/signup, `(protected)/` group for authenticated pages

## ID and Time Encoding

- All IDs are strings in JSON (`type: string` in the spec) to avoid JS BigInt issues with int64 values
- All timestamps are ISO 8601 strings (`format: date-time`) - the frontend uses `new Date(iso)` directly

## Document Workflow States

Pending / InReview / Approved | Rejected | ChangesRequested (resubmit back to InReview)
