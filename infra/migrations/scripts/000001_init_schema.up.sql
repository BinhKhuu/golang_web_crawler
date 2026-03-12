CREATE TABLE raw_data (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    content_type TEXT,
    raw_content TEXT NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    raw_data_id INT REFERENCES raw_data(id) ON DELETE SET NULL,
    url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    priority INT NOT NULL DEFAULT 0,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
