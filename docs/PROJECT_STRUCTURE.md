# Project Structure

```
agnos-middleware/
├── cmd/
│   ├── server/main.go          # entry point: load config, connect DB, wire dependencies, graceful shutdown
│   └── mockhis/main.go         # fake Hospital A HIS for local development / demos
├── internal/
│   ├── config/                 # environment-variable configuration + validation
│   ├── model/                  # domain types (Hospital, Staff, Patient), search filter, sentinel errors
│   ├── auth/                   # JWT issue / verify (TokenManager)
│   ├── repository/             # persistence: interfaces + GORM/PostgreSQL implementations
│   ├── hospitalapi/            # HTTP client for a hospital's HIS (GET /patient/search/{id})
│   ├── service/                # business rules (AuthService, PatientService) - no HTTP, no SQL
│   ├── middleware/             # Gin middleware: RequireAuth (JWT)
│   ├── handler/                # HTTP layer: validation, status codes, router
│   └── apidocs/                # OpenAPI spec (openapi.yaml) + Swagger UI at /swagger
├── migrations/001_init.sql     # schema + seed hospitals (run by postgres on first start)
├── dev/mock_his.sql            # dev only: points hospital-a at the mock-his container
├── nginx/nginx.conf            # reverse proxy + rate limit on /staff/*
├── docs/                       # this documentation
├── Dockerfile                  # multi-stage build -> small non-root image (server + mockhis)
├── docker-compose.yml          # nginx + app + postgres (+ mock-his for development)
└── .env.example
```

## Layering and dependency direction

```
HTTP request
   │
   ▼
nginx ──► handler ──► service ──► repository ──► PostgreSQL
              │           └────► hospitalapi ──► Hospital HIS (HTTP; mock-his in development)
              └── middleware (JWT)
```

* **handler** only parses/validates input and maps errors to HTTP status codes.
* **service** holds the rules (hospital scoping, caching of HIS results, credential checks).
* **repository** and **hospitalapi** are Go interfaces, so services and handlers are unit-tested with
  in-memory fakes - no database or network needed.
* Dependencies point inward only: `handler → service → repository/hospitalapi → model`.

## Docker Compose services

| Service | Image | Exposed | Role |
|---|---|---|---|
| nginx | nginx:1.27-alpine | port 80 (host) | reverse proxy, rate limit on `/staff/*` |
| app | built from `Dockerfile` | internal :8080 | Go/Gin API |
| db | postgres:16-alpine | internal :5432 | data; runs `migrations/001_init.sql` on first start |
| mock-his | built from `Dockerfile` | internal :8081 | fake Hospital A HIS (development only) |

## Key design decisions

| Decision | Reason |
|---|---|
| Hospital scope comes from the **JWT**, never from the request body | A staff member cannot search another hospital by editing the request |
| `patients.hospital_id` + every query filtered by it | Tenant boundary enforced at the data layer |
| Login returns one generic error for unknown hospital / user / wrong password | Prevents username enumeration |
| `/patient/search` is **POST** with a JSON body | National IDs / names / phone numbers stay out of URLs and access logs |
| HIS result is cached into `patients` (upsert on `hospital_id, patient_hn`) | Later multi-criteria searches (name, DOB, ...) can find the patient |
| HIS outage degrades to local results (logged as a warning) | A third-party outage should not take the search down |
| HIS base URL stored per hospital (`hospitals.api_base_url`) | New hospitals / a mock HIS need no code change |
| bcrypt for passwords, HS256 JWT with expiry | Standard, no plain-text secrets stored |

## Testing

`go test ./... -cover` - no database or network needed. Every API has positive and negative cases:
handlers (`internal/handler`), services (`internal/service`), JWT middleware, HIS client (against
`httptest` servers) and the mock HIS (exercised through the real HIS client).
