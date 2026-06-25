# Clinics API

Go (Gin + GORM) backend for the clinics project — a design-first migration of a
Laravel/Backpack admin. The HTTP contract is defined in
[`api/openapi.yaml`](api/openapi.yaml) and implemented with `oapi-codegen`
strict-server handlers.

> **Frontend developer? Start here** — pick your stack:
> - **Angular** → [docs/frontend-api-guide.md](docs/frontend-api-guide.md)
>   (OpenAPI Generator + RxJS services + HTTP interceptor)
> - **React** → [docs/frontend-react-api-guide.md](docs/frontend-react-api-guide.md)
>   (Orval + TanStack Query + Ant Design — setup & patterns)
> - **React feature build** → [docs/frontend-patient-dashboard-guide.md](docs/frontend-patient-dashboard-guide.md)
>   (login, patient create, and the 5-tab patient management dashboard)
>
> Both cover client generation from the spec, auth wiring, and consuming the
> generated code against this API's pagination + error conventions.

## API reference

When the server is running, two always-current references are served directly:

- **Swagger UI:** `/docs` — interactive, generated from the live spec
- **Raw spec:** `/openapi.json` — import into Postman/Insomnia or a client generator
- **Health:** `/healthz`

## Run locally

Configuration is read from `.env` (see `internal/config`). The server defaults to
the port in `HTTP_PORT` (or a platform-injected `$PORT`).

```bash
make run            # build + run
# or
go run ./cmd/api
```

Common targets: `make build`, `make gen` (regenerate the API client from the
spec), `make vet`, `make test`. See the [Makefile](Makefile).

## Layout

```
api/                 OpenAPI spec + the Vercel serverless function entrypoint
cmd/api/             Local / container server entrypoint (main)
internal/app/        Shared bootstrap: config -> DB -> router
internal/api/        Generated server interface + hand-written handlers
internal/config/     Env loading, MySQL DSN, TLS
internal/models/     GORM models
docs/                Developer guides (frontend integration, etc.)
```

## Database TLS

The MySQL connection supports `DB_SSLMODE` of `""` (off), `true` (system trust
store), `verify-ca`, or `verify-full`. For managed providers with a private CA
(e.g. Aiven), set `DB_SSLMODE=verify-ca` and provide the CA via
`DB_SSLROOTCERT` — either a file path or the inline PEM.
