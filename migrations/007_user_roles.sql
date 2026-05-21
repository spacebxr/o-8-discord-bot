CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT PRIMARY KEY,
    role_ids TEXT[] NOT NULL DEFAULT '{}',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT timezone('utc'::text, now()) NOT NULL
);
