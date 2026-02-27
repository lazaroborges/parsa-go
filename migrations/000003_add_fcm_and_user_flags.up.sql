-- Migration 000003: Add FCM notifications and user flow flags

-- Add new columns to users
ALTER TABLE public.users ADD COLUMN has_finished_openfinance_flow boolean DEFAULT false NOT NULL;
ALTER TABLE public.users ADD COLUMN trigger_swipe_cards_flow boolean DEFAULT false NOT NULL;

-- FCM device tokens table
CREATE TABLE public.fcm_device_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    token character varying(255) NOT NULL,
    device_type character varying(10) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    last_used timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT device_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT device_tokens_token_unique UNIQUE (token),
    CONSTRAINT device_tokens_device_type_check CHECK (((device_type)::text = ANY ((ARRAY['ios'::character varying, 'android'::character varying])::text[]))),
    CONSTRAINT device_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_fcm_device_tokens_user_id ON public.fcm_device_tokens USING btree (user_id);
CREATE INDEX idx_fcm_device_tokens_token ON public.fcm_device_tokens USING btree (token);

-- FCM notification preferences table
CREATE TABLE public.fcm_notification_preferences (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    budgets_enabled boolean DEFAULT true NOT NULL,
    general_enabled boolean DEFAULT true NOT NULL,
    accounts_enabled boolean DEFAULT true NOT NULL,
    transactions_enabled boolean DEFAULT true NOT NULL,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT notification_preferences_pkey PRIMARY KEY (id),
    CONSTRAINT notification_preferences_user_id_key UNIQUE (user_id),
    CONSTRAINT notification_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

-- FCM notifications table
CREATE TABLE public.fcm_notifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    title character varying(255) NOT NULL,
    message text NOT NULL,
    category character varying(50) NOT NULL,
    data jsonb DEFAULT '{}'::jsonb,
    opened_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT notifications_pkey PRIMARY KEY (id),
    CONSTRAINT notifications_category_check CHECK (((category)::text = ANY ((ARRAY['accounts'::character varying, 'budgets'::character varying, 'general'::character varying, 'transactions'::character varying])::text[]))),
    CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_fcm_notifications_user_id ON public.fcm_notifications USING btree (user_id);
CREATE INDEX idx_fcm_notifications_created_at ON public.fcm_notifications USING btree (created_at DESC);
