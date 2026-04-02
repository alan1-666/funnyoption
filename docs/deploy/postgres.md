# PostgreSQL Bootstrap

## Local or remote container

Use the compose file at `/Users/zhangza/code/funnyoption/deploy/postgres/docker-compose.yml`.

```bash
docker compose -f deploy/postgres/docker-compose.yml up -d
```

Default bootstrap values:

- database: `funnyoption`
- user: `funnyoption`
- password: `funnyoption`
- port: `5432`

If the target server already has a PostgreSQL container, you can also reuse it and create a separate database, which is what the current test server setup uses.

## Apply schema

```bash
export FUNNYOPTION_POSTGRES_DSN='postgres://funnyoption:funnyoption@127.0.0.1:5432/funnyoption?sslmode=disable'
./scripts/apply_migrations.sh
```

## Test environment example

See `/Users/zhangza/code/funnyoption/configs/test/funnyoption.env.example`.
