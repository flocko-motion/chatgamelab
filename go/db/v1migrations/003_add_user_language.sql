-- Migration: Add language preference to users
-- Adds a language column to app_user table for storing user's preferred language (ISO 639-1 code)

ALTER TABLE app_user
ADD COLUMN language text NOT NULL DEFAULT 'en';

COMMENT ON COLUMN app_user.language IS 'User''s preferred language (ISO 639-1 code: en, de, fr, etc.)';
