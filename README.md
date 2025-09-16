## Import Files Service (Go)

Centralized import pipeline that reads files from blob storage and inserts validated records as JSONB into per-customer Postgres DBs. Offers REST, gRPC, and CLI interfaces, with jobs/logs stored centrally.

### Features
- REST: POST `/enqueue` to queue imports
- gRPC: `importer.Importer/Enqueue` using `Struct` request
- CLI: enqueue and run workers
- Background workers with goroutines and concurrency
- Central tables: `import_jobs`, `import_logs`
- Per-customer target table equals product type (`users`, `organizations`, `courses`) with `data JSONB`

### Config
- Preferred: JSON config via `CONFIG_PATH` and `CONFIG_ENV` (e.g., `default`, `docker`). See `config.json`.
- Env fallback (if no JSON): `APP_DB_DSN`, `REST_ADDR`, `GRPC_ADDR`, `WORKER_CONCURRENCY`, `CUSTOMER_MAP_PATH`.

### Project Structure
```
cmd/
  rest/         # REST server (POST /enqueue)
  grpc/         # gRPC server (importer.Importer/Enqueue)
  cli/          # CLI to enqueue and run workers
internal/
  blob/         # Blob interface and file:// implementation
  config/       # Env config and customer map loader
  db/           # App DB (jobs/logs) and Customer DB (JSONB inserts)
  grpcsvc/      # Manual gRPC service descriptor and handler
  importer/     # Orchestration: read->parse->validate->insert
  jobs/         # Job repository (enqueue/poll/complete/fail/log)
  parser/       # CSV/TSV/KV parsing (XLSX stub)
  products/     # Product types and target table mapping
  validate/     # Minimal schema validation
Dockerfile
docker-compose.yml
customer_map.json  # sample customerId->DSN mapping
```

### Docker Compose
```bash
docker compose up --build
```

REST will listen on `localhost:8080`. gRPC on `localhost:9090`.
Compose passes `CONFIG_PATH=/app/config.json` and `CONFIG_ENV=docker`.

### Run Without Docker
1) Start a Postgres you can reach, then set env:
```bash
export CONFIG_PATH=./config.json
export CONFIG_ENV=default
```
2) Start REST server (spawns workers):
```bash
go run ./cmd/rest
```
3) Or start gRPC server:
```bash
go run ./cmd/grpc
```
4) Or use CLI (also runs workers in-foreground):
```bash
go run ./cmd/cli --customer customer1 --product users --file ./sample/users.csv
```

### REST Example
```bash
curl -X POST localhost:8080/enqueue \
  -H 'Content-Type: application/json' \
  -d '{"customer_id":"customer1","product_type":"users","blob_uri":"file:///data/users.csv"}'
```

### CLI Example
```bash
APP_DB_DSN=postgres://app:app@localhost:5432/app?sslmode=disable \
go run ./cmd/cli --customer customer1 --product users --file ./sample/users.csv
```

### Notes
- CSV/TSV and simple key=value text supported. XLSX stubbed for now.
- Extend `internal/blob` for cloud blobs (S3/Azure/GCS).
- Product schemas enforce required fields only for brevity.


