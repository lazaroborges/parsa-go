-- Migration 000005: One token per user; add updated_at to fcm_device_tokens

-- Remove duplicate tokens per user, keeping the most recently used (or most recently created)
DELETE FROM public.fcm_device_tokens a
USING public.fcm_device_tokens b
WHERE a.user_id = b.user_id
  AND a.id != b.id
  AND (a.last_used < b.last_used OR (a.last_used = b.last_used AND a.created_at < b.created_at));

-- Add unique constraint on user_id (one token per user)
ALTER TABLE public.fcm_device_tokens ADD CONSTRAINT device_tokens_user_id_unique UNIQUE (user_id);

-- Drop token-unique constraint (token can be overwritten per user)
ALTER TABLE public.fcm_device_tokens DROP CONSTRAINT IF EXISTS device_tokens_token_unique;

-- Add updated_at (when token/device_type changed; distinct from last_used)
ALTER TABLE public.fcm_device_tokens ADD COLUMN updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL;
