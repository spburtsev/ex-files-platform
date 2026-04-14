# Architecture

## Project Overview

Ex-Files is a document notarization & approval platform. Users organize work into **workspaces / issues / documents**, with versioning, review workflows, and an append-only audit log. Three roles: root, manager, employee.

## Monorepo Structure

- **protocol/** Protocol Buffer definitions, generates types for both backend and frontend
- **ex-files-backend/** Go HTTP API (Gin, GORM, PostgreSQL, MinIO)
- **ex-files-frontend/** SvelteKit 5 app (Svelte 5, Tailwind CSS 4, TypeScript, bun)

## Protocol-Driven Communication

All API types are defined in `protocol/` as `.proto` files. `make proto` generates:
- Go types in `ex-files-backend/gen/`
- TypeScript types in `ex-files-frontend/src/lib/gen/`

Backend handlers serialize responses as Protobuf binary. Frontend deserializes using `@bufbuild/protobuf` (`fromBinary(Schema, bytes)`).

## Backend Layers

- **handlers/** - HTTP handlers, route logic, role checks, audit logging
- **services/** - Repository interfaces + GORM implementations, token service, bcrypt hasher, MinIO storage
- **models/** - GORM models with auto-migration on startup
- **middleware/** - JWT auth middleware (extracts user from cookie or Bearer token)
- **seed/** - Creates default users on startup

No separate service/business-logic layer. Handlers call repositories directly.
Services implement interfaces defined in `services/interfaces.go` to implement dependency injection. Mock repositories are used for unit-testing.

## Frontend Patterns

- **Remote functions**: `src/lib/data.remote.ts` (queries via `query()`) and `src/lib/commands.remote.ts` (mutations via `command()`) - these are SvelteKit server functions that proxy to the backend
- **Proto utils**: `src/lib/proto-utils.ts` - helpers like `protoTsToDate()`, `formatTimestamp()`, `isManager()`, `initials()`
- **i18n**: Paraglide JS with `en` and `pl` locales. Messages in `messages/{locale}.json`. Detection: URL - cookie - base locale
- **UI components**: shadcn-svelte (bits-ui) in `src/lib/components/ui/`
- **Auth**: Session cookie (`session`, HTTP-only, 8h TTL). Server hook in `hooks.server.ts` validates and redirects unauthenticated users
- **Routing**: `(auth)/` group for login/signup, `(protected)/` group for authenticated pages

## Document Workflow States

Pending / InReview / Approved | Rejected | ChangesRequested (resubmit back to InReview)
