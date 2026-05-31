# Migrations

We use Goose.

Files are stored in service-specific directories under `migrations/`.

Current directories:

* `migrations/auth`
* `migrations/booking`

## Rules

* Each file has `Up` and `Down`
* Do not edit applied migrations
* One change = one file
