# FunnyOption Staging Deployment

This folder contains the first server-oriented staging deployment assets.

## Files

- [docker-compose.staging.yml](/Users/zhangza/code/funnyoption/deploy/staging/docker-compose.staging.yml)
- [funnyoption.xyz.conf](/Users/zhangza/code/funnyoption/deploy/staging/funnyoption.xyz.conf)

## Expected env source

Copy:

- [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)

to a real runtime file named `.env.staging` next to the compose file on the
server.

## Typical server workflow

```bash
docker compose --env-file .env.staging -f deploy/staging/docker-compose.staging.yml --profile ops run --rm migrate
docker compose --env-file .env.staging -f deploy/staging/docker-compose.staging.yml up -d --build
```

