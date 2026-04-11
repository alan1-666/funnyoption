-- Run once on the primary after Patroni cluster is healthy (psql as postgres superuser).
-- Replace passwords before applying, or use variables in your shell.

-- FunnyOption application user + DB
CREATE USER funnyoption WITH PASSWORD 'replace-funnyoption-password';
CREATE DATABASE funnyoption OWNER funnyoption;
GRANT ALL PRIVILEGES ON DATABASE funnyoption TO funnyoption;

-- Wallet SaaS application user + DB
CREATE USER wallet WITH PASSWORD 'replace-wallet-password';
CREATE DATABASE wallet OWNER wallet;
GRANT ALL PRIVILEGES ON DATABASE wallet TO wallet;
