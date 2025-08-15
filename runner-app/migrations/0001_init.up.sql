-- Jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    jobspec_id VARCHAR(255) UNIQUE NOT NULL,
    jobspec_data JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'queued',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Executions table
CREATE TABLE IF NOT EXISTS executions (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES jobs(id),
    provider_id VARCHAR(255) NOT NULL,
    region VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    output_data JSONB,
    receipt_data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Diffs table
CREATE TABLE IF NOT EXISTS diffs (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES jobs(id),
    region_a VARCHAR(100) NOT NULL,
    region_b VARCHAR(100) NOT NULL,
    similarity_score DECIMAL(5,4),
    diff_data JSONB,
    classification VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Outbox table
CREATE TABLE IF NOT EXISTS outbox (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    published_at TIMESTAMP
);
