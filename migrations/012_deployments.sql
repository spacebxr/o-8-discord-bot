CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message TEXT NOT NULL,
    host_id TEXT NOT NULL DEFAULT '',
    co_host_id TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'scheduled',
    discord_message_id TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_seconds BIGINT DEFAULT 0,
    announced_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deployment_participants (
    deployment_id UUID REFERENCES deployments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    PRIMARY KEY (deployment_id, user_id)
);
