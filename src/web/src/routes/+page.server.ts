import type { PageServerLoad, Actions } from './$types';
import type { EnergyReading } from '$lib/types';

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:8080';

export const load: PageServerLoad = async () => {
  try {
    const now = new Date();
    const from = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    const params = new URLSearchParams({
      from: from.toISOString(),
      to: now.toISOString(),
      bucket: '120',
      limit: '720',
    });

    const res = await fetch(`${API_BASE}/api/v1/energy/history?${params}`);
    if (!res.ok) return { history: [] as EnergyReading[] };

    const history: EnergyReading[] = await res.json();
    return { history };
  } catch {
    return { history: [] as EnergyReading[] };
  }
};

export const actions: Actions = {
  testNotification: async ({ fetch: serverFetch }) => {
    try {
      const res = await serverFetch(`${API_BASE}/api/v1/notifications/test`, {
        method: 'POST',
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        return { success: false, error: body.message ?? 'Failed to send notification' };
      }
      return { success: true };
    } catch {
      return { success: false, error: 'Could not reach the API' };
    }
  },
};
