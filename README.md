# University Room Booking

## How to run

The canonical project deployment lives at `deploy/` (collective, in progress).

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

## Development

See CONTRIBUTING.md for workflow and branching rules.
