# University Room Booking

## How to run

Run the integrated local stack from the repository root:

```bash
docker compose up --build
```

This brings up auth-service, booking-service, gateway-api, frontend,
PostgreSQL for each service, RabbitMQ, and goose migration sidecars.
After migrations booking-service always fetches https://table.nsu.ru once,
imports rooms, expands room lessons into concrete schedule rows, and then
starts serving HTTP.
Endpoints:

- Frontend → http://localhost:3000
- API Gateway → http://localhost:8080
- Booking PostgreSQL → localhost:5432 (booking / booking)
- Auth PostgreSQL → localhost:5433 (auth / auth)
- RabbitMQ management → http://localhost:15672 (guest / guest)

For booking-service development in isolation:

```bash
docker compose -f cmd/booking/compose.yaml up --build
```

This brings up PostgreSQL, RabbitMQ, a goose migration sidecar, and the
booking-service itself. Endpoints:

- Booking API → http://localhost:8080
- RabbitMQ management → http://localhost:15672 (guest / guest)

`GET /rooms` and `GET /rooms/{id}` are anonymous. `POST /booking` and
`DELETE /booking/{id}` require a `Bearer` JWT signed with the same
`JWT_SECRET` that booking-service uses (compose defaults to a dev-only
placeholder — rotate it for any real deployment, and align it with
auth-service's signing key when integrating). The token must carry the
claims `sub` (user id as string), `login`, and `role` (`student_b`,
`student_m`, `student_a`, `teacher`, or `admin`).

## Parser import

The parser is part of booking-service startup, not a separate runtime service.
It always runs once before booking-service starts serving HTTP. Its import
window can be controlled with environment variables:

- `PARSER_BASE_URL` — NSU timetable base URL, defaults to `https://table.nsu.ru`.
- `PARSER_WEEKS_AHEAD` — how many weeks of recurring timetable rows to materialize, defaults to `16`.
- `PARSER_TIMEZONE` — timezone used for concrete schedule dates, defaults to `Asia/Novosibirsk`.

Imported lessons are stored in `bookings` as `creator_role=admin` and
`user_id=0`. On every parser run previous parser rows are replaced; existing
user-created bookings are preserved. If an imported lesson overlaps an existing
booking, that lesson is skipped and counted in the import log.

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

## Development

See CONTRIBUTING.md for workflow and branching rules.
