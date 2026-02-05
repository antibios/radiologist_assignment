CREATE TABLE radiologists (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    max_concurrent_studies INTEGER NOT NULL DEFAULT 5,
    credentials TEXT[], -- Array of strings
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE shifts (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    work_type TEXT NOT NULL,
    sites TEXT[],
    priority_level INTEGER NOT NULL DEFAULT 0,
    required_credentials TEXT[],
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE roster_entries (
    id BIGSERIAL PRIMARY KEY,
    shift_id BIGINT NOT NULL REFERENCES shifts(id),
    radiologist_id TEXT NOT NULL REFERENCES radiologists(id),
    start_date TIMESTAMP NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE assignment_rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    priority_order INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    condition_filters JSONB, -- Stored as JSON
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE assignments (
    id BIGSERIAL PRIMARY KEY,
    study_id TEXT NOT NULL,
    radiologist_id TEXT NOT NULL REFERENCES radiologists(id),
    shift_id BIGINT NOT NULL REFERENCES shifts(id),
    strategy TEXT NOT NULL,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW()
);
