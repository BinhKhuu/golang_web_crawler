# 🕷️ Golang Web Crawler

A web crawler built with Go.

---

## 🛠️ Required Tools

### Core
| Tool | Version | Install |
|------|---------|---------|
| [Go](https://golang.org/dl/) | 1.25+ | `brew install go` |
| [Node.js](https://nodejs.org/) | 18+ | `brew install node` |
| [npm](https://www.npmjs.com/) | 9+ | Comes with Node.js |

### Infrastructure
| Tool | Version | Install |
|------|---------|---------|
| [Docker](https://docs.docker.com/get-docker/) | 24+ | `brew install --cask docker` |
| [Docker Compose](https://docs.docker.com/compose/) | 2+ | Included with Docker Desktop |

### Database
| Tool | Version | Install |
|------|---------|---------|
| [PostgreSQL](https://www.postgresql.org/download/macosx/) | 18+ | `brew install postgresql@18` |
| [golang-migrate](https://github.com/golang-migrate/migrate) | latest | `brew install golang-migrate` |

### Environment
| Tool | Version | Install |
|------|---------|---------|
| [godotenv](https://github.com/joho/godotenv) | latest | Comes with Go (used via `github.com/joho/godotenv` package) |

### AI/LLM
| Tool | Version | Install |
|------|---------|---------|
| [Ollama](https://ollama.ai) | latest | `brew install ollama` |

---

## ⚙️ Environment Configuration (.env)

Create a `.env` file in the project root with the following settings:

```bash
# Database (PostgreSQL via Docker)
DB_USER=myuser
DB_PASSWORD=mypassword
DB_HOST=localhost
DB_PORT=5433
DB_NAME=jobs_webcrawler
DB_SSLMODE=disable

# Test Configuration
RUN_LLM_TESTS=1        # Set to 1 to enable LLM tests
RUN_FETCH_TESTS=1      # Set to 1 to enable fetch tests

# Crawler Configuration (optional)
# CRAWLER_MAX_DEPTH=5
# CRAWLER_ALLOWED_DOMAINS=seek.com.au,example.com,iana.org
```

### Timezone

**Recommended:** Set your system and PostgreSQL timezone to **UTC (UTC+0)** to avoid timestamp inconsistencies. All timestamps in this project are expected to be in UTC. Use the helper in [`internal/typeutil/time.go`](internal/typeutil/time.go) for getting the current time — it always returns UTC.

### Export Environment Variables

Before running migrations or the application, export the environment variables:

```bash
export $(grep -v '^#' .env | xargs)
```

---

## 🐳 PostgreSQL Setup

### Start the Database

Start PostgreSQL using Docker Compose:

```bash
docker-compose up -d postgres
```

The database will be available at `localhost:5433`.

### Install PostgreSQL Locally (Optional)

If you prefer to run PostgreSQL locally instead of Docker:

```bash
brew install postgresql@18
brew services start postgresql@18
```

Then update the `.env` file with your local PostgreSQL credentials.

---

## 🔄 Database Migrations

### Install golang-migrate

```bash
brew install golang-migrate
```

### Running Migrations

After exporting environment variables (see `.env` section above):

```bash
migrate -path infra/migrations/scripts \
  -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" \
  up
```

### Creating a New Migration

```bash
migrate create -ext sql -dir infra/migrations/scripts -seq <migration_name>
```

For more details, see [`MIGRATIONS.md`](MIGRATIONS.md).

---

## 🤖 Ollama Setup

### Install Ollama

```bash
brew install ollama
```

### Start Ollama Service

```bash
# Terminal 1: Start the Ollama server
ollama serve
```

### Install and Run Models

```bash
# Terminal 2: Run a model
ollama run mistral:latest

# List installed models
ollama list
```

### Available Models

| Model | Description |
|-------|-------------|
| `gemma2` | Fastest model with consistent results (recommended) |
| `mistral:latest` | Second fastest for data extraction |
| `qwen3.5:latest` | Slowest but most capable, resource intensive |

### Configure Model in Application

Edit [`internal/llm/llm.go`](internal/llm/llm.go:24) to change the model:

```go
const (
    Model        = "mistral:latest"
    MaxMemoryMBs = 16384
)
```

### Ollama in Docker (Optional)

Ollama can also be run in Docker by uncommenting the `ollama` service in [`docker-compose.yaml`](docker-compose.yaml:22).

For more details, see [`OLLAMA.md`](OLLAMA.md).

---

## 🐛 Visual Studio Code Debug Setup

### Debug Profile Configuration

Create a `.vscode/launch.json` file in the project root with the following configuration:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Scheduler",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/scheduler",
            "envFile": "${workspaceFolder}/.env",
            "args": [],
            "showLog": true
        },
        {
            "name": "Debug API Server",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/api",
            "envFile": "${workspaceFolder}/.env",
            "args": [],
            "showLog": true
        }
    ]
}
```

### Using the Debug Profile

1. Open the file you want to debug (e.g., `cmd/scheduler/main.go`)
2. Set breakpoints by clicking on the line numbers
3. Press `F5` or go to Run → Start Debugging
4. Select the appropriate debug configuration from the dropdown
5. The application will start with the `.env` file loaded

---

## 🚀 Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/your-username/golangwebcrawler.git
cd golangwebcrawler
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install Playwright dependencies
go get github.com/playwright-community/playwright-go
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
```

### 3. Configure Environment

Copy the example environment file and update values:

```bash
# Create .env file (see ⚙️ Environment Configuration section above)
```

### 4. Start Services

```bash
# Start PostgreSQL
docker-compose up -d postgres

# Run database migrations
export $(grep -v '^#' .env | xargs)
migrate -path infra/migrations/scripts \
  -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" \
  up
```

### 5. Run the Application

```bash
# Run the scheduler (orchestrates crawl + parse jobs)
go run cmd/scheduler/main.go

# Or use the debug profile in VS Code (see 🐛 Visual Studio Code Debug Setup)
```

---

## 📁 Project Structure & Root Detection

### Project Root Marker

The project root is identified by the presence of a `.project-root` marker file at the repository root. This explicit marker is used by [`internal/env/env.go`](internal/env/env.go) to detect the project boundary.

**Detection Logic:**
1. Checks current working directory for `.project-root`
2. Searches upward from current directory to parent directories
3. Falls back to searching from the calling package's location

### Key Files

| File | Purpose |
|------|---------|
| [`.project-root`](.project-root) | Marker file identifying the project root directory |
| [`go.mod`](go.mod) | Go module definition and dependencies |
| [`.env`](.env) | Environment variables (not tracked in Git) |
| [`docker-compose.yaml`](docker-compose.yaml) | Docker services configuration (PostgreSQL, Ollama) |
| [`.golangci.yaml`](.golangci.yaml) | Go linter configuration |
| [`.gitignore`](.gitignore) | Git ignore rules |

---

## 📚 Additional Documentation

- [`MIGRATIONS.md`](MIGRATIONS.md) - Database migration guide
- [`OLLAMA.md`](OLLAMA.md) - Ollama LLM setup guide
- [`DOCKER.md`](DOCKER.md) - Docker configuration guide
- [`TESTING.md`](TESTING.md) - Testing guide

---

## 🐛 playwright package

playwright is used to 'smart crawl' - installation requires installing all the playwright dependencies:

```bash
go get github.com/playwright-community/playwright-go
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
```

* Playwright Go driver v1.57.0 installs (version number may vary)
