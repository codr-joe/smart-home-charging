import { readable, writable } from 'svelte/store';
import type { EnergyReading } from '$lib/types';

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';
const WS_BASE = API_BASE.replace(/^http/, 'ws');

function createEnergyStream() {
  let socket: WebSocket | null = null;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  let backoff = 1000;

  const { subscribe, set } = writable<EnergyReading | null>(null);

  function connect() {
    socket = new WebSocket(`${WS_BASE}/api/v1/stream`);

    socket.onmessage = (event) => {
      try {
        const reading: EnergyReading = JSON.parse(event.data);
        set(reading);
        backoff = 1000;
      } catch {
        // ignore malformed frames
      }
    };

    socket.onclose = () => {
      reconnectTimeout = setTimeout(() => {
        backoff = Math.min(backoff * 2, 30000);
        connect();
      }, backoff);
    };

    socket.onerror = () => {
      socket?.close();
    };
  }

  function start() {
    connect();
    return () => {
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
      socket?.close();
    };
  }

  return { subscribe, start };
}

export const energyStream = createEnergyStream();

export async function fetchHistory(from: Date, to: Date, limit = 500): Promise<EnergyReading[]> {
  const params = new URLSearchParams({
    from: from.toISOString(),
    to: to.toISOString(),
    limit: String(limit),
  });
  const res = await fetch(`${API_BASE}/api/v1/energy/history?${params}`);
  if (!res.ok) throw new Error(`Failed to fetch history: ${res.status}`);
  return res.json();
}
