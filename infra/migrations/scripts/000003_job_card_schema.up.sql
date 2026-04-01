CREATE TABLE extracted_jobdata(
    id SERIAL PRIMARY KEY, 
    
    title TEXT NOT NULL,
    company TEXT NOT NULL,
    location TEXT,
    
    -- Using TEXT for salary as it often contains ranges like "$120k - $150k"
    salary TEXT, 
    
    -- Descriptions can be long, so TEXT is safer than VARCHAR
    description TEXT, 
    
    -- Single link field to match ExtractedJobData.Link
    link TEXT,
    
    -- Store skills as comma-separated string or JSONB array
    skills TEXT,
    
    -- Track when the row was created
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster searching by company or title
CREATE INDEX idx_extracted_jobdata_company ON extracted_jobdata(company);
CREATE INDEX idx_extracted_jobdata_title ON extracted_jobdata(title);
CREATE INDEX idx_extracted_jobdata_link ON extracted_jobdata(link);