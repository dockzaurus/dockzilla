# PostgreSQL with pg_cron, which PgQue needs for its ticker and maint jobs.
#
# The official image ships no pg_cron and no package manager path to it: the
# alpine variants build PostgreSQL from source into /usr/local, while Alpine's
# postgresql-pg_cron installs under /usr/lib/postgresql18 and drags a second,
# complete PostgreSQL in as a dependency. The server would never look there.
#
# So the builder stage installs the package only to harvest two paths from it,
# and is then discarded along with the duplicate PostgreSQL. Copying across
# prefixes is safe because both halves are the same major version built against
# the same libc, which is what the extension ABI is keyed on.
#
# Enabling it takes more than this image; see the postgres command in
# docker-compose.yml.

FROM postgres:18-alpine AS pgcron
RUN apk add --no-cache postgresql-pg_cron

FROM postgres:18-alpine
COPY --from=pgcron /usr/lib/postgresql18/pg_cron.so /usr/local/lib/postgresql/
COPY --from=pgcron /usr/share/postgresql18/extension/pg_cron* /usr/local/share/postgresql/extension/
