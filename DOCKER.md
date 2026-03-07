# Docker Setup

This project uses Docker Compose to run the required infrastructure services (PostgreSQL and Redis).

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed
- [Docker Compose](https://docs.docker.com/compose/install/) installed

---

## Services

| Service    | Image           | Port   |
|------------|-----------------|--------|
| PostgreSQL | postgres:18.0   | 5432   |
| Redis      | redis:8.6       | 6379   |

---

## Getting Started

### 1. Configure Environment Variables

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

### 2. Start All Services
``` bash
docker compose up -d
```

### 3. Verify Services Are running
``` bash
docker compose ps
```

### 4. View Logs
``` bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f postgres
docker compose logs -f redis
```

### 5. Stop Services

``` bash
docker compose down
```
remove persisted volums

``` bash
docker compose down -v
```

# Connecting to Services

### PostgreSQL
connect to PostgreSQL
``` bash
psql postgres://myuser:mypassword@localhost:5433/jobs_webcrawler
```

### Redis
``` bash
docker compose exec redis redis-cli
```

# Infrastructure Compose
An additional Compose file is available under docker-compose.yml for extended infrastructure services (e.g. migrations).

To use it:

``` bash
docker compose -f infra/docker-compose.yml up 
```


# Debug / Trouble shooting

## Can't connect to PostgreSQL

check if another process is listening to the port 5433

```
lsof -i :5433
```

You’ll see something like:

    postgres → your local instance is using the port

    docker-proxy or com.docker.backend → your Docker instance is using the port

This tells you exactly which server your psql command is hitting.
Default port is 5432 this project deliberarly uses 5433 to avoid clashes