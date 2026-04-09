import { readable, writable } from 'svelte/store';
import type { EnergyReading } from '$lib/types';

// Used for REST API calls. In production the app and API share the same origin
// (same-origin routing via the Gateway), so the base URL is empty and relative
// paths are used. In local development set VITE_API_BASE_URL=http://localhost:8080
// in src/web/.env to reach the API on a different port.
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '';

// Derives the WebSocket base URL from the current page's origin at runtime so
// it works in any environment without build-time configuration.
function getWsUrl(path: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}${path}`;
}

function createEnergyStream() {
  let socket: WebSocket | null = null;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  let backoff = 1000;

  const { subscribe, set } = writable<EnergyReading | null>(null);

  function connect() {
    socket = new WebSocket(getWsUrl('/api/v1/stream'));

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
