CREATE TABLE IF NOT EXISTS role_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id_a VARCHAR(255) NOT NULL,
    role_id_b VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role_id_a, role_id_b)
);
