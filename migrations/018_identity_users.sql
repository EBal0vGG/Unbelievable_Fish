CREATE TABLE IF NOT EXISTS identity_users (
    user_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES identity_companies(company_id),
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_identity_users_company_id
    ON identity_users (company_id);
