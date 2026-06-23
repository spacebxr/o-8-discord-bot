CREATE TABLE IF NOT EXISTS personnel_stats (
    user_id TEXT PRIMARY KEY,
    total_messages BIGINT DEFAULT 0,
    deployments_participated BIGINT DEFAULT 0,
    last_message_at TIMESTAMP WITH TIME ZONE
);
