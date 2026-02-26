-- Rollback migration 000003

DROP TABLE IF EXISTS public.fcm_notifications;
DROP TABLE IF EXISTS public.fcm_notification_preferences;
DROP TABLE IF EXISTS public.fcm_device_tokens;

ALTER TABLE public.users DROP COLUMN IF EXISTS trigger_swipe_cards_flow;
ALTER TABLE public.users DROP COLUMN IF EXISTS has_finished_openfinance_flow;
