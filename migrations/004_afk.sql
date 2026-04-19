CREATE TABLE IF NOT EXISTS afk_status (
    user_id TEXT PRIMARY KEY,
    reason TEXT NOT NULL,
    since TIMESTAMP WITH TIME ZONE DEFAULT timezone('utc'::text, now()) NOT NULL
);
