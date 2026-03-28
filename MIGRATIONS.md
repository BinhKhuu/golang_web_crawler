package go-migrate

### Creating your first migration
You start by generating a versioned pair of files. These files represent the forward change and the rollback.

```sh
migrate create -ext sql -dir infra/migrations/scripts -seq init_schema    
```

This produces something like:

```
infra/migrations/scripts/
  000001_init_schema.up.sql
  000001_init_schema.down.sql
```

Each file is empty until you add your SQL.

---

### Writing the migration SQL
The `.up.sql` file contains the schema change you want to apply. For example:

```sql
CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

The `.down.sql` file reverses it:

```sql
DROP TABLE jobs;
```

This pairing ensures your schema can move forward and backward cleanly.

---

### Running the migration against your Docker database
Use the same connection string you tested earlier. For example:

```sh
migrate -path infra/migrations/scripts \
  -database "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable" \
  up
```
If everything is correct, you’ll see output confirming the migration was applied. The schema version is tracked inside the database so future migrations apply in order.

---

### Running migrations from your Go application
If you want your service to apply migrations automatically at startup, you can embed the library:

```go
m, err := migrate.New(
    "file://infra/migrations/scripts",
    "postgres://user:password@localhost:5432/jobs_webcrawler?sslmode=disable",
)
if err != nil {
    log.Fatal(err)
}

err = m.Up()
if err != nil && err != migrate.ErrNoChange {
    log.Fatal(err)
}
```

## .env credentials
In the .env file add

```
DB_USER=myuser
DB_PASSWORD=mypassword
DB_HOST=localhost
DB_PORT=5433
DB_NAME=jobs_webcrawler
DB_SSLMODE=disable
```

export the env values

``` sh
export $(grep -v '^#' .env | xargs)
```

run migration command
``` sh
migrate -path infra/migrations/scripts \
    -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" \
    up
```

## Dirty Migrations
A migration becomes dirty when:

- the `.up.sql` started running,
- something failed partway,
- but the migration version was already written into `schema_migrations`.

This leaves the database in a state where:

- the schema does **not** match the migration version,
- migrate refuses to run `up` or `down`,
- and you must intervene manually.

---

## How to clear a dirty migration (two valid strategies)

### 1) Development workflow: reset and re-run
This is the simplest and safest when you don’t care about existing data.

- Drop the database (or drop all tables + `schema_migrations`)
- Recreate it
- Run `migrate up` again

This guarantees a clean state and avoids manual repair.

This is the standard approach during early development because it avoids chasing partial schema changes.

---

### 2) Production workflow: repair the broken migration
In production you **cannot** drop the database, so you must:

1. Identify the migration version that failed (e.g., version 1).
2. Inspect the `.up.sql` file for that version.
3. Manually apply the missing SQL inside `psql` until the schema matches what the migration *should* have created.
4. Mark the migration as clean:

```
migrate force <version>
```
for example if first migration fails
``` sh
migrate -path infra/migrations/scripts  \
   -database "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable" \
  force 1
```
This tells migrate:

> “The schema now matches version X. Continue from here.” it will execute migrations after x not x itself

This is the correct way to fix a dirty migration in production because it preserves data and ensures the schema and migration history stay aligned.

#### Run migrations in order 
Migrations are designed to be run sequentially because:

1. Dependencies — migration 000002 likely depends on schema created in 000001. Running them out of order breaks those dependencies.

2. Data loss — running a down on an older migration while newer ones are applied can drop columns/tables that newer migrations rely on.

3. Dirty state — the migration tool tracks the current version as a single number. Running out of order confuses the version tracking and can leave the database in a dirty state.

The safe workflow is always:
```
000001 up → 000002 up → 000003 up 
000003 down → 000002 down → 000001 down 
```
If you need to fix a specific migration, the correct approach is to:

Roll back down to that migration sequentially
Fix the migration file
Run up again sequentially


#### Migration downs
migrate down by default only steps down 1 migration at a time.

migrate ... down → rolls back 1 migration (your current version)
migrate ... down 2 → rolls back 2 migrations
migrate ... drop → drops everything (dangerous, no steps)
So if you're on version 000001, running down will only execute 000001_init_schema.down.sql. A 000002 down would only run if you were on version 000002 or passed a steps argument that covers it.

#### Migration target specific migration
specific number
```sh
migrate -path infra/migrations/scripts \
  -database "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable" \
  up 1
```

```sh
migrate -path infra/migrations/scripts \
  -database "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable" \
  down 1
```