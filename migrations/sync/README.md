# migrations/sync

Postgres migrations for the `sync` schema. Owned by the `sync` service.

Files are numbered sequentially: `001_`, `002_`, etc.
Applied by `golang-migrate` on service startup.

Each service only applies migrations in its own directory.
No migration file ever touches another service's schema.

See DOMAIN_MODEL.md for the entity definitions that these migrations implement.
