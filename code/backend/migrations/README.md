# Migrations

One pair of files per change, named `YYYYMMDDHHMMSS_what_it_does.up.sql` and `YYYYMMDDHHMMSS_what_it_does.down.sql`.

- The server applies every `*.up.sql` it has not applied yet, in filename order, on boot and before it listens.
  What has been applied is recorded in the `schema_migrations` table, so running again is a no-op.
- Each file runs in one transaction together with its `schema_migrations` row: a failing file changes nothing.
- A file whose SQL contains `CONCURRENTLY` (an index on a table that already has rows) runs OUTSIDE a transaction,
  because Postgres refuses it inside one. Such a file holds only plain statements separated by `;` (no `$$` bodies),
  each written to be safe to run twice (`CREATE INDEX CONCURRENTLY IF NOT EXISTS ...`).
- `*.down.sql` is for people and tooling, never run by the server. Every `.up.sql` has one.
- Never edit a file that has been merged: add a new pair.
- The first real pair arrives with the story "Site data from the database"; the scaffold ships none on purpose.
