CREATE TABLE job_cards (
    -- Using the external ID from the site as the Primary Key
    id SERIAL PRIMARY KEY, 
    
    title TEXT NOT NULL,
    company TEXT NOT NULL,
    location TEXT,
    
    -- Using TEXT for salary as it often contains ranges like "$120k - $150k"
    salary TEXT, 
    
    -- Descriptions can be long, so TEXT is safer than VARCHAR
    description TEXT, 
    
    -- URLs can exceed 255 chars, use TEXT
    url TEXT NOT NULL,
    link TEXT,
    classification TEXT,
    
    -- TIMESTAMPTZ handles timezones automatically
    update_date TIMESTAMPTZ,
    scrape_date TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Track when the row itself was last modified in your DB
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster searching by company or title
CREATE INDEX idx_job_company ON job_cards(company);
CREATE INDEX idx_job_title ON job_cards(title);
