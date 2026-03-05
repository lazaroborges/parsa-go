-- Rollback migration 000005

ALTER TABLE public.fcm_device_tokens DROP COLUMN IF EXISTS updated_at;

ALTER TABLE public.fcm_device_tokens DROP CONSTRAINT IF EXISTS device_tokens_user_id_unique;

ALTER TABLE public.fcm_device_tokens ADD CONSTRAINT device_tokens_token_unique UNIQUE (token);
