CREATE TABLE IF NOT EXISTS identity_companies (
    company_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    inn TEXT NOT NULL,
    ogrn TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
