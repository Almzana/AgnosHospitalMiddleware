-- Schema for the Hospital Middleware system.
-- Executed automatically by the postgres container on first start
-- (mounted into /docker-entrypoint-initdb.d).

CREATE TABLE IF NOT EXISTS hospitals (
    id           BIGSERIAL PRIMARY KEY,
    code         VARCHAR(50)  NOT NULL UNIQUE,   -- identifier used by staff APIs, e.g. "hospital-a"
    name         VARCHAR(255) NOT NULL,
    api_base_url VARCHAR(255) NOT NULL DEFAULT '', -- HIS base URL; empty = no external HIS
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS staff (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    hospital_id   BIGINT       NOT NULL REFERENCES hospitals (id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- the same username may exist in different hospitals
    CONSTRAINT uq_staff_username_hospital UNIQUE (username, hospital_id)
);

CREATE INDEX IF NOT EXISTS idx_staff_hospital ON staff (hospital_id);

-- Patient mirrors the fields returned by the Hospital Information System (HIS)
-- plus hospital_id, which is what enforces "staff only see their own hospital".
CREATE TABLE IF NOT EXISTS patients (
    id             BIGSERIAL PRIMARY KEY,
    hospital_id    BIGINT       NOT NULL REFERENCES hospitals (id),
    first_name_th  VARCHAR(100) NOT NULL DEFAULT '',
    middle_name_th VARCHAR(100) NOT NULL DEFAULT '',
    last_name_th   VARCHAR(100) NOT NULL DEFAULT '',
    first_name_en  VARCHAR(100) NOT NULL DEFAULT '',
    middle_name_en VARCHAR(100) NOT NULL DEFAULT '',
    last_name_en   VARCHAR(100) NOT NULL DEFAULT '',
    date_of_birth  DATE,
    patient_hn     VARCHAR(50)  NOT NULL,        -- hospital number (HN)
    national_id    VARCHAR(20)  NOT NULL DEFAULT '',
    passport_id    VARCHAR(20)  NOT NULL DEFAULT '',
    phone_number   VARCHAR(30)  NOT NULL DEFAULT '',
    email          VARCHAR(255) NOT NULL DEFAULT '',
    -- '' = HIS did not send a gender (same convention as the other optional columns).
    -- VARCHAR, not CHAR: CHAR(1) would pad '' to ' ' and the API would return a space.
    gender         VARCHAR(1)   NOT NULL DEFAULT '' CHECK (gender IN ('M', 'F', '')),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- HN is only unique inside one hospital
    CONSTRAINT uq_patient_hospital_hn UNIQUE (hospital_id, patient_hn)
);

CREATE INDEX IF NOT EXISTS idx_patients_hospital_national ON patients (hospital_id, national_id) WHERE national_id <> '';
CREATE INDEX IF NOT EXISTS idx_patients_hospital_passport ON patients (hospital_id, passport_id) WHERE passport_id <> '';
CREATE INDEX IF NOT EXISTS idx_patients_hospital_dob      ON patients (hospital_id, date_of_birth);

-- Seed data
INSERT INTO hospitals (code, name, api_base_url) VALUES
    ('hospital-a', 'Hospital A', 'https://hospital-a.api.co.th'),
    ('hospital-b', 'Hospital B', '')
ON CONFLICT (code) DO NOTHING;
