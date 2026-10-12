# API Specification

Base URL (through nginx): `http://localhost` - all bodies are `application/json`.
Interactive docs (Swagger UI): `http://localhost/swagger` - spec in `internal/apidocs/openapi.yaml`.
Errors always look like `{"error": "<message>"}`.

---

## GET /health
Liveness check. → `200 {"status":"ok"}`

---

## POST /staff/create
Create a hospital staff member.

**Request**

| Field | Type | Rules |
|---|---|---|
| username | string | required, 3-100 chars |
| password | string | required, 8-72 chars |
| hospital | string | required, hospital `code` (seeded: `hospital-a`, `hospital-b`) |

```json
{ "username": "alice", "password": "s3cret-pass", "hospital": "hospital-a" }
```

**Responses**

| Status | When | Body |
|---|---|---|
| 201 | created | `{"id":1,"username":"alice","hospital":"hospital-a"}` |
| 400 | invalid JSON / validation failed / unknown hospital | `{"error":"..."}` |
| 409 | username already exists in that hospital | `{"error":"staff already exists in this hospital"}` |
| 500 | unexpected server error | `{"error":"internal server error"}` |

---

## POST /staff/login
Authenticate and receive a JWT. Same request body as `/staff/create`.

**Responses**

| Status | When | Body |
|---|---|---|
| 200 | success | `{"token":"<jwt>","expires_at":"2026-10-09T10:00:00Z"}` |
| 400 | invalid JSON / validation failed | `{"error":"..."}` |
| 401 | unknown hospital, unknown user **or** wrong password (deliberately indistinguishable) | `{"error":"invalid username, password or hospital"}` |
| 500 | unexpected server error | `{"error":"internal server error"}` |

Token claims: `staff_id`, `hospital_id`, `username`, `iat`, `exp` (HS256).

---

## POST /patient/search
Search patients **of the caller's own hospital**. Requires `Authorization: Bearer <token>`.
The hospital is taken from the token; any hospital sent in the body is ignored.

**Request** - every field optional; supplied fields are combined with AND.

| Field | Type | Matching |
|---|---|---|
| national_id | string | exact |
| passport_id | string | exact |
| first_name | string | contains, case-insensitive, Thai **or** English name |
| middle_name | string | contains, case-insensitive, Thai or English |
| last_name | string | contains, case-insensitive, Thai or English |
| date_of_birth | string `YYYY-MM-DD` | exact |
| phone_number | string | exact |
| email | string | exact, case-insensitive |

An empty body / `{}` returns the hospital's patients (max 100).

```json
{ "national_id": "1234567890123", "date_of_birth": "1990-05-17" }
```

**Behaviour:** local database first. If `national_id` or `passport_id` was supplied and nothing matches
locally, the hospital's HIS is called (`GET {api_base_url}/patient/search/{id}`), the result is stored for
this hospital, and the filter is applied again.

**Responses**

| Status | When |
|---|---|
| 200 | success (possibly empty list) |
| 400 | invalid JSON, wrong field type, or `date_of_birth` not `YYYY-MM-DD` |
| 401 | missing / malformed / invalid / expired token |
| 500 | unexpected server error |

```json
{
  "count": 1,
  "patients": [
    {
      "first_name_th": "สมชาย", "middle_name_th": "", "last_name_th": "ใจดี",
      "first_name_en": "Somchai", "middle_name_en": "", "last_name_en": "Jaidee",
      "date_of_birth": "1990-05-17",
      "patient_hn": "HN123",
      "national_id": "1234567890123",
      "passport_id": "",
      "phone_number": "0812345678",
      "email": "somchai@example.com",
      "gender": "M"
    }
  ]
}
```

---

## Upstream: Hospital A HIS (consumed, not exposed)

`GET https://hospital-a.api.co.th/patient/search/{id}` - `id` is a national_id or passport_id.
Returns the patient JSON shown above (without the `count`/`patients` wrapper). `404` = patient not found.
The base URL is stored per hospital in `hospitals.api_base_url`, so other hospitals can be added without code changes.

---

## Quick try (curl)

```bash
curl -X POST localhost/staff/create -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"s3cret-pass","hospital":"hospital-a"}'

TOKEN=$(curl -s -X POST localhost/staff/login -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"s3cret-pass","hospital":"hospital-a"}' | jq -r .token)

curl -X POST localhost/patient/search -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"national_id":"1234567890123"}'
```
