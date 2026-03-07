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

