# Agnos Hospital Middleware

Go + Gin middleware that lets hospital staff log in and search patients **of their own hospital**,
pulling patient data from the hospital's HIS API when it is not yet known locally.

Stack: Go 1.26 · Gin · GORM/PostgreSQL · JWT · Docker Compose · Nginx

## Run

```bash
cp .env.example .env        # change JWT_SECRET / POSTGRES_PASSWORD
docker compose up --build   # nginx on http://localhost (HTTP_PORT), app + postgres internal
```

The schema and two seed hospitals (`hospital-a`, `hospital-b`) are created on first start from
`migrations/001_init.sql`. To reset the database: `docker compose down -v`.

### Mock HIS

`docker compose` also starts `mock-his` ([cmd/mockhis](cmd/mockhis/main.go)), a fake Hospital A HIS, and
[dev/mock_his.sql](dev/mock_his.sql) points `hospital-a` at it. Log in as a `hospital-a` staff member and
search by one of these ids:

| id | Result |
|---|---|
| `1234567890123` | Somchai Jaidee (M) |
| `1103700012345` | Somsri Manee Rakdee (F) |
| `AA1234567` (passport) | John Smith (M) |
| `3100500099999` | Mana Odthon (HIS sends no gender) |
| `9999999999999` | HIS returns 500 → search still answers 200 with an empty list |
| anything else | HIS returns 404 → empty list |

## Develop / test

```bash
go mod tidy          # first time: downloads dependencies and creates go.sum (commit it)
go test ./... -cover
```

Unit tests need no database or network: repositories and the HIS client are interfaces replaced by
in-memory fakes; the HIS HTTP client is tested against `httptest` servers.

## Docs

* [Project structure & design decisions](docs/PROJECT_STRUCTURE.md)
* [API specification](docs/API_SPEC.md)
* Swagger UI: http://localhost/swagger (spec: [internal/apidocs/openapi.yaml](internal/apidocs/openapi.yaml))
* [ER diagram](docs/ER_DIAGRAM.md)

## Configuration

| Variable | Default | Description |
|---|---|---|
| DATABASE_URL | - (required) | PostgreSQL DSN |
| JWT_SECRET | - (required, ≥16 chars) | HS256 signing key |
| PORT | 8080 | listen port |
| TOKEN_TTL | 24h | token lifetime |
| HIS_TIMEOUT | 5s | timeout for HIS calls |
