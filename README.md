# University Room Booking

## How to run

Run the integrated local stack from the repository root:

```bash
cp .env.example .env
```

Fill in `.env`: replace secret placeholders, keep database credentials and
connection URLs consistent, and URL-encode special characters in URL passwords.
Then run:

```bash
make build-auth VERSION=v1
make build-booking VERSION=v1
make build-gateway VERSION=v1
docker compose build
docker compose up -d --no-build
```

This brings up auth-service, booking-service, gateway-api, frontend,
PostgreSQL for each service, RabbitMQ, and goose migration sidecars.
After migrations booking-service imports the NSU timetable in the background
only when no imported schedule exists in the database.
Endpoints:

- Frontend → http://localhost:3000
- API Gateway → http://localhost:8080
- Booking PostgreSQL → localhost:5432 (credentials from `.env`)
- Auth PostgreSQL → localhost:5433 (credentials from `.env`)
- RabbitMQ management → http://localhost:15672 (credentials from `.env`)

Compose reads the root `.env` for interpolation; it is ignored by Git.
Only `.env.example` is committed. Internal service names and ports are part
of this local deployment. `NEXT_PUBLIC_API_GATEWAY_URL` is public and baked
into the frontend at build time; never put secrets in `NEXT_PUBLIC_*`.
Changing PostgreSQL credentials does not update existing database volumes;
existing databases require matching credentials or an explicit credential update.

For booking-service development in isolation:

```bash
cp cmd/booking/.env.example cmd/booking/.env
# Fill in cmd/booking/.env before running:
make build-booking VERSION=v1
docker compose --env-file cmd/booking/.env -f cmd/booking/compose.yaml build booking
docker compose --env-file cmd/booking/.env -f cmd/booking/compose.yaml up -d --no-build
```

This brings up PostgreSQL, RabbitMQ, a goose migration sidecar, and the
booking-service itself. Endpoints:

- Booking API → http://localhost:8080
- RabbitMQ management → http://localhost:15672 (credentials from the standalone env)

`GET /rooms` and `GET /rooms/{id}` are anonymous. `POST /booking` and
`DELETE /booking/{id}` require a `Bearer` JWT signed with the same
`JWT_SECRET` that booking-service uses (set it in `.env` and align it with
auth-service's signing key when integrating). The token must carry the
claims `sub` (user id as string), `login`, and `role` (`student_b`,
`student_m`, `student_a`, `teacher`, or `admin`).

## Build, Release, Run

Each Go service has independent versions: `AUTH_VERSION`, `BOOKING_VERSION`,
and `GATEWAY_VERSION` in `.env`. Binaries are stored as
`build/<service>/<version>/<binary>` and are ignored by Git. Building a new
version keeps older versions until an explicit `make clean`.

Build targets run ordinary `go build` in the Go container defined in
`compose.build.yaml`. Docker Desktop must be running on macOS. Docker chooses
the native architecture automatically; no GOOS, GOARCH or CGO_ENABLED overrides
are used. The builder and runtime images both use Debian 12, including runtime
libraries for normally built Go binaries. Artifacts are saved in the local
`build/` directory. Runtime Dockerfiles only copy those artifacts. Frontend
keeps its existing Docker build process.

To update auth only:

```bash
make build-auth VERSION=v2
# Set AUTH_VERSION=v2 in .env; leave other service versions unchanged.
docker compose build auth-service
docker compose up -d --no-build --no-deps auth-service
```

Build creates binaries and packages images. Release selects service versions
and runtime configuration in `.env`. Run starts those images with `--no-build`.
Treat published version labels as immutable: use a new version for changed code.

To roll auth back, set `AUTH_VERSION=v1` and run the same `up --no-build --no-deps`
command. Keep the old image locally; if it was removed, package the preserved
`build/auth/v1/auth` with `docker compose build auth-service` without recompiling.

## Parser import

The parser is part of booking-service startup, not a separate runtime service.
It runs when no imported schedule exists in the database. Its import
window can be controlled with environment variables:

- `PARSER_BASE_URL` — NSU timetable base URL, defaults to `https://table.nsu.ru`.
- `PARSER_WEEKS_AHEAD` — how many weeks of recurring timetable rows to materialize, defaults to `16`.
- `PARSER_TIMEZONE` — timezone used for concrete schedule dates, defaults to `Asia/Novosibirsk`.

Startup imports use a PostgreSQL advisory lock. Only one booking-service
instance checks and imports at a time; waiting instances recheck the database
after acquiring the lock and skip parsing if the schedule already exists.
The lock is released on completion, failure, or loss of the database connection.
No additional service or migration is required. Parser status remains local
to each instance; a waiting instance reports `running` until its check finishes.

Imported lessons are stored in `bookings` as `creator_role=admin` and
`user_id=0`. On every parser run previous parser rows are replaced; existing
user-created bookings are preserved. If an imported lesson overlaps an existing
booking, that lesson is skipped and counted in the import log.

## Shutdown

Auth, booking, and gateway handle SIGTERM/SIGINT, stop accepting connections,
and wait for active HTTP requests up to `SHUTDOWN_TIMEOUT` (default `5s`).
After the timeout, remaining connections are closed. Booking also cancels
and waits for its background import before closing PostgreSQL and RabbitMQ.
Docker allows `STOP_GRACE_PERIOD` (default `30s`) before forcibly killing
the process. Keep it longer than `SHUTDOWN_TIMEOUT`, with time for cleanup.

## Tests

Regular checks:

```bash
go test ./...
cd frontend && npm run lint && npm run build
```

Parser import integration tests require Docker/testcontainers:

```bash
go test -tags=integration -run 'TestPostgresStore_ReplaceParsedSchedule' ./internal/adapter/booking/postgres
```

Expected behavior: testcontainers starts a temporary PostgreSQL, goose applies
booking migrations, parser rows are imported into `rooms`/`bookings`, a second
import replaces old parser rows, and overlaps with user bookings are skipped
without deleting user data.

Concurrency and shutdown checks:

```bash
go test -race ./internal/platform/httpx ./internal/core/parser/service
go test -tags=integration -race -count=1 -timeout=5m -run 'TestScheduleImportLock' ./internal/adapter/booking/postgres
```

Expected behavior: six simultaneous startup attempts import only once;
a failed import releases the lock, and a waiting instance can be canceled.
HTTP tests verify that an active request finishes during graceful shutdown
and that a stalled request is disconnected after the shutdown timeout.

## Development

`timetable-homework-tgbot` has its own Go module and Compose deployment.
Its `.env` remains separate from the root application configuration
and are ignored by Git. Its Telegram credentials are not needed by the main stack.

See CONTRIBUTING.md for workflow and branching rules.
