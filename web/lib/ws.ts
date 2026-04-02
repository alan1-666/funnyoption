function toWsUrl(base: string) {
  if (base.startsWith("https://")) return base.replace("https://", "wss://");
  if (base.startsWith("http://")) return base.replace("http://", "ws://");
  return base;
}

export function getWsBaseUrl() {
  return toWsUrl(process.env.NEXT_PUBLIC_WS_BASE_URL ?? "http://127.0.0.1:8081");
}

export function createWsUrl(path: string) {
  return `${getWsBaseUrl()}${path}`;
}
