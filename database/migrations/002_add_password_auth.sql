-- Migration: Add password authentication support
ALTER TABLE users ALTER COLUMN google_sub DROP NOT NULL;
ALTER TABLE users ADD COLUMN password_hash TEXT;
