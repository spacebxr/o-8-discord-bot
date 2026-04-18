CREATE TABLE IF NOT EXISTS stopwatches (
    user_id TEXT PRIMARY KEY,
    start_time TIMESTAMP WITH TIME ZONE,
    total_seconds BIGINT DEFAULT 0 NOT NULL
);
