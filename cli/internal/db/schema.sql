-- CloudSync Database Schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE
);

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(64) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(50),
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB
);

-- API tokens table
CREATE TABLE IF NOT EXISTS api_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100),
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used TIMESTAMP WITH TIME ZONE,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

-- Sync folders table
CREATE TABLE IF NOT EXISTS folders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    folder_id VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    encryption_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    encryption_key_id VARCHAR(255)
);

-- Device folder mappings
CREATE TABLE IF NOT EXISTS device_folders (
    id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    folder_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    local_path VARCHAR(1024) NOT NULL,
    sync_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sync_direction VARCHAR(20) NOT NULL DEFAULT 'bidirectional',
    exclude_patterns TEXT[], -- Array of glob patterns to exclude
    last_sync_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    UNIQUE (device_id, folder_id)
);

-- File versions table
CREATE TABLE IF NOT EXISTS file_versions (
    id SERIAL PRIMARY KEY,
    folder_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    relative_path VARCHAR(1024) NOT NULL,
    version_id VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    hash VARCHAR(64) NOT NULL,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    device_id INTEGER REFERENCES devices(id),
    mime_type VARCHAR(255),
    metadata JSONB,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (folder_id, relative_path, version_id)
);

-- Sync events table
CREATE TABLE IF NOT EXISTS sync_events (
    id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    folder_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    file_version_id INTEGER REFERENCES file_versions(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL, -- created, modified, deleted, conflict, etc.
    relative_path VARCHAR(1024) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    details JSONB
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_file_versions_folder_path ON file_versions(folder_id, relative_path);
CREATE INDEX IF NOT EXISTS idx_sync_events_device_folder ON sync_events(device_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_device_folders_device_id ON device_folders(device_id);
CREATE INDEX IF NOT EXISTS idx_device_folders_folder_id ON device_folders(folder_id);
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);

-- Add function for updating updated_at timestamp
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add triggers for tables that need automatic updated_at
CREATE TRIGGER update_users_timestamp
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_devices_timestamp
BEFORE UPDATE ON devices
FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_folders_timestamp
BEFORE UPDATE ON folders
FOR EACH ROW EXECUTE FUNCTION update_timestamp(); 