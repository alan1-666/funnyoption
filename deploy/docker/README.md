# Container Builds

These Dockerfiles are intended to be built from the repository root:

```bash
docker build -f deploy/docker/api.Dockerfile -t funnyoption-api .
docker build -f deploy/docker/matching.Dockerfile -t funnyoption-matching .
docker build -f deploy/docker/account.Dockerfile -t funnyoption-account .
docker build -f deploy/docker/ledger.Dockerfile -t funnyoption-ledger .
docker build -f deploy/docker/settlement.Dockerfile -t funnyoption-settlement .
docker build -f deploy/docker/chain.Dockerfile -t funnyoption-chain .
docker build -f deploy/docker/ws.Dockerfile -t funnyoption-ws .
docker build -f deploy/docker/web.Dockerfile -t funnyoption-web .
docker build -f deploy/docker/admin.Dockerfile -t funnyoption-admin .
```

## Runtime notes

- Go services read their configuration from runtime environment variables such as:
  - `FUNNYOPTION_POSTGRES_DSN`
  - `FUNNYOPTION_KAFKA_BROKERS`
  - `FUNNYOPTION_*_GRPC_ADDR`
  - `FUNNYOPTION_API_HTTP_ADDR`
  - `FUNNYOPTION_WS_HTTP_ADDR`
  - chain / vault settings
- `web` and `admin` embed `NEXT_PUBLIC_*` values at build time, so production API / RPC / chain values should be passed as `--build-arg`.
- `admin` also needs:
  - `FUNNYOPTION_DEFAULT_OPERATOR_USER_ID`
  - `FUNNYOPTION_OPERATOR_WALLETS`

## Example frontend build

```bash
docker build \
  -f deploy/docker/web.Dockerfile \
  -t funnyoption-web \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api.example.com \
  --build-arg NEXT_PUBLIC_WS_BASE_URL=https://ws.example.com \
  --build-arg NEXT_PUBLIC_ADMIN_BASE_URL=https://admin.example.com \
  --build-arg NEXT_PUBLIC_CHAIN_ID=56 \
  --build-arg NEXT_PUBLIC_CHAIN_NAME="BNB Smart Chain" \
  --build-arg NEXT_PUBLIC_CHAIN_RPC_URL=https://bsc-dataseed.binance.org \
  --build-arg NEXT_PUBLIC_COLLATERAL_SYMBOL=USDT \
  .
```

## Example backend run

```bash
docker run --rm \
  --env-file .env.production \
  -p 8080:8080 \
  funnyoption-api
```

## Staging on BSC Testnet

For the current recommended staging shape, see:

- [configs/staging/funnyoption.env.example](/Users/zhangza/code/funnyoption/configs/staging/funnyoption.env.example)
- [docs/deploy/staging-bsc-testnet.md](/Users/zhangza/code/funnyoption/docs/deploy/staging-bsc-testnet.md)
