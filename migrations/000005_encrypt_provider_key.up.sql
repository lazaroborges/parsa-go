-- Modify provider_key column to TEXT to accommodate encrypted values
-- Encrypted values are larger due to AES-256-GCM (nonce + ciphertext + auth tag) + base64 encoding
ALTER TABLE users
    ALTER COLUMN provider_key TYPE TEXT;

-- Note: This migration does not encrypt existing plaintext provider_key values.
-- If you have existing data, you must either:
-- 1. Clear existing provider_key values: UPDATE users SET provider_key = NULL;
-- 2. Run a data migration script with access to ENCRYPTION_KEY to re-encrypt existing values
--
-- New values inserted/updated through the application will be automatically encrypted.
