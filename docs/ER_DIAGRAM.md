# ER Diagram

```mermaid
erDiagram
    HOSPITALS ||--o{ STAFF    : employs
    HOSPITALS ||--o{ PATIENTS : "treats / owns"

    HOSPITALS {
        bigint      id PK
        varchar     code UK "e.g. hospital-a"
        varchar     name
        varchar     api_base_url "HIS base URL, empty = none"
        timestamptz created_at
    }

    STAFF {
        bigint      id PK
        varchar     username
        varchar     password_hash "bcrypt"
        bigint      hospital_id FK
        timestamptz created_at
    }

    PATIENTS {
        bigint      id PK
        bigint      hospital_id FK
        varchar     first_name_th
        varchar     middle_name_th
        varchar     last_name_th
        varchar     first_name_en
        varchar     middle_name_en
        varchar     last_name_en
        date        date_of_birth
        varchar     patient_hn "hospital number"
        varchar     national_id
        varchar     passport_id
        varchar     phone_number
        varchar     email
        varchar     gender "M, F or empty"
        timestamptz created_at
        timestamptz updated_at
    }
```

## Constraints and indexes

| Table | Constraint / index | Purpose |
|---|---|---|
| hospitals | `UNIQUE (code)` | staff APIs identify a hospital by code |
| staff | `UNIQUE (username, hospital_id)` | same username may exist in different hospitals |
| staff | `FK hospital_id → hospitals.id` | a staff member belongs to exactly one hospital |
| patients | `UNIQUE (hospital_id, patient_hn)` | HN is unique only within a hospital; target of the upsert |
| patients | `FK hospital_id → hospitals.id` | tenant boundary |
| patients | `CHECK gender IN ('M','F','')` | matches the HIS contract; `''` = HIS sent no gender |
| patients | partial index `(hospital_id, national_id) WHERE national_id <> ''` | fast id lookup |
| patients | partial index `(hospital_id, passport_id) WHERE passport_id <> ''` | fast id lookup |
| patients | index `(hospital_id, date_of_birth)` | DOB search |

## Mapping of HIS response → `patients`

The columns mirror the Hospital A response one-to-one (`first_name_th … gender`), so a HIS payload can be
stored without transformation. `hospital_id` is the only added column - it is what makes
"staff can only search patients of their own hospital" enforceable.
