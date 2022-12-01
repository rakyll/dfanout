CREATE DATABASE dfanout;

CREATE TABLE IF NOT EXISTS endpoints (
    fanout_name VARCHAR(1024) NOT NULL,
    endpoint_name VARCHAR(1024) NOT NULL,
    is_primary BOOLEAN,
    http_endpoint JSON,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY(fanout_name, endpoint_name)
);

CREATE INDEX IF NOT EXISTS idx_endpoints_fanout_name_primary ON endpoints(fanout_name, is_primary);

CREATE INDEX IF NOT EXISTS idx_endpoints_is_primary ON endpoints(is_primary);