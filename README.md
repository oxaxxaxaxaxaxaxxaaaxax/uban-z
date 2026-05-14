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

## Development

See CONTRIBUTING.md for workflow and branching rules.
