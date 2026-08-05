# Vendored PgQue schema

`pgque.sql` is a **verbatim copy of upstream** — [NikolayS/pgque](https://github.com/NikolayS/pgque),
`sql/pgque.sql` at the tag in `VERSION`. It is not ours. Do not edit it.

PgQue publishes no package: it is not on PGXN, not a C extension with an OS
package, and `pgque-go` is a driver that embeds no schema and has no migrate
command. The installer is a file in a git repo, so the choice is when to copy
it, not whether. Copying it once, here, is what gives us a pinned version, a
readable diff on upgrade, and a deploy that needs nothing but this repo.

## Files

| File               | What it is                                                   |
| ------------------ | ------------------------------------------------------------ |
| `pgque.sql`        | The upstream installer, verbatim                              |
| `pgque.sql.sha256` | Checksum of the above, so an edit or a truncated fetch fails loudly |
| `VERSION`          | The upstream tag this copy came from                          |

## Usage

```sh
task backend:pgque:verify    # does the working copy still match its checksum?
task backend:pgque:install   # apply it to DATABASE_URL, then assert pgque.version()
task backend:pgque:vendor    # re-fetch the pinned version (e.g. after a bad merge)
```

## Upgrading

```sh
task backend:pgque:vendor -- v0.3.0
```

That rewrites all three files. Review the `pgque.sql` diff like any other
dependency bump — that diff is the whole point of vendoring, and the filename
stays fixed so git can produce one. Then bump `pgque-go` in `go.mod` to the
matching tag so client and schema stay in lockstep.

The installer is idempotent: re-running it upgrades an existing install in
place, so `task backend:pgque:install` is both the install and the upgrade path.

## What does not belong here

Anything of ours. Queue creation, grants to our roles, and any local tweak go in
`scripts/sql` as ordinary migrations that run after this file.

## Operational note

`pgque.start()` is **not** called by the install task, because it requires
`pg_cron`, which our Postgres image does not have. Without it, four jobs that
`start()` would have scheduled have to be driven by the application instead:
`ticker_loop()`, `maint_retry_events()` (~30s), `maint()` (~30s), and
`maint_rotate_tables_step2()` (~10s). See `pkg/queue/pgqueue`.
