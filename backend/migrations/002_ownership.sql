-- funnyoption ownership and grants

ALTER SCHEMA public OWNER TO funnyoption;
GRANT ALL ON SCHEMA public TO funnyoption;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER TABLE public.%I OWNER TO funnyoption', r.tablename);
    END LOOP;

    FOR r IN
        SELECT sequencename
        FROM pg_sequences
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER SEQUENCE public.%I OWNER TO funnyoption', r.sequencename);
    END LOOP;
END $$;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO funnyoption;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO funnyoption;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL PRIVILEGES ON TABLES TO funnyoption;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL PRIVILEGES ON SEQUENCES TO funnyoption;
