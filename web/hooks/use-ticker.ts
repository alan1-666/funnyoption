"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import { createWsUrl } from "@/lib/ws";

export interface TickerSnapshot {
  lastPrice: number;
  bestBid: number;
  bestAsk: number;
  updatedAt: number;
}

export interface LiveTicker {
  yes: TickerSnapshot | null;
  no: TickerSnapshot | null;
  connected: boolean;
}

const EMPTY: LiveTicker = { yes: null, no: null, connected: false };

interface TickerStore {
  state: LiveTicker;
  listeners: Set<() => void>;
  refCount: number;
  sockets: WebSocket[];
}

const stores = new Map<number, TickerStore>();

function getOrCreateStore(marketId: number): TickerStore {
  let store = stores.get(marketId);
  if (store) return store;

  store = {
    state: EMPTY,
    listeners: new Set(),
    refCount: 0,
    sockets: [],
  };
  stores.set(marketId, store);
  return store;
}

function emit(store: TickerStore) {
  store.listeners.forEach((fn) => fn());
}

function startSockets(marketId: number, store: TickerStore) {
  for (const outcome of ["YES", "NO"] as const) {
    const url = createWsUrl(`/ws?stream=ticker&book_key=${marketId}:${outcome}`);
    const ws = new WebSocket(url);

    ws.onopen = () => {
      store.state = { ...store.state, connected: true };
      emit(store);
    };

    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as {
          last_price: number;
          best_bid: number;
          best_ask: number;
          occurred_at_millis: number;
        };
        const snap: TickerSnapshot = {
          lastPrice: data.last_price,
          bestBid: data.best_bid,
          bestAsk: data.best_ask,
          updatedAt: data.occurred_at_millis,
        };
        const key = outcome === "YES" ? "yes" : "no";
        store.state = { ...store.state, [key]: snap };
        emit(store);
      } catch {
        /* ignore */
      }
    };

    ws.onclose = () => {
      store.state = { ...store.state, connected: false };
      emit(store);
    };

    store.sockets.push(ws);
  }
}

function stopSockets(store: TickerStore) {
  store.sockets.forEach((ws) => ws.close());
  store.sockets = [];
}

export function useTicker(marketId: number): LiveTicker {
  const [, setTick] = useState(0);

  useEffect(() => {
    const store = getOrCreateStore(marketId);
    store.refCount++;

    if (store.refCount === 1) {
      startSockets(marketId, store);
    }

    const listener = () => setTick((n) => n + 1);
    store.listeners.add(listener);

    return () => {
      store.listeners.delete(listener);
      store.refCount--;
      if (store.refCount <= 0) {
        stopSockets(store);
        stores.delete(marketId);
      }
    };
  }, [marketId]);

  return stores.get(marketId)?.state ?? EMPTY;
}
