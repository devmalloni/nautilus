BEGIN;

CREATE TABLE hook_definitions (
    id TEXT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    payload_scheme JSONB,
    http_request_method VARCHAR(10) NOT NULL,
    total_attempts INT NOT NULL
);

CREATE TABLE hook_configurations (
    id TEXT PRIMARY KEY,
    tag VARCHAR(255) NOT NULL,
    hook_definition_id TEXT NOT NULL REFERENCES hook_definitions(id) ON DELETE CASCADE,
    url VARCHAR(255) NOT NULL,
    client_secret TEXT NOT NULL,
    client_rsa_private_key TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (tag, hook_definition_id)
);

CREATE TABLE hook_schedules (
    id TEXT PRIMARY KEY,
    hook_configuration_id TEXT NOT NULL REFERENCES hook_configurations(id) ON DELETE CASCADE,
    http_request_method VARCHAR(10) NOT NULL,
    url VARCHAR(255) NOT NULL,
    payload JSONB,
    status VARCHAR(50) NOT NULL,
    max_attempt INT NOT NULL,
    current_attempt INT NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE hook_executions (
    id TEXT PRIMARY KEY,
    hook_schedule_id TEXT NOT NULL REFERENCES hook_schedules(id) ON DELETE CASCADE,
    response_status INT,
    response_payload TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMIT;