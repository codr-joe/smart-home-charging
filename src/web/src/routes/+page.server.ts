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

    const [historyRes, settingsRes, currentRes] = await Promise.all([
      fetch(`${API_BASE}/api/v1/energy/history?${params}`),
      fetch(`${API_BASE}/api/v1/notifications/settings`),
      fetch(`${API_BASE}/api/v1/energy/current`),
    ]);

    const history: EnergyReading[] = historyRes.ok ? await historyRes.json() : [];
    const notificationsEnabled: boolean = settingsRes.ok
      ? (await settingsRes.json()).enabled
      : null;
    const currentReading: EnergyReading | null = currentRes.ok ? await currentRes.json() : null;

    return { history, notificationsEnabled, currentReading };
  } catch {
    return { history: [] as EnergyReading[], notificationsEnabled: null, currentReading: null };
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

  toggleNotifications: async ({ fetch: serverFetch, request }) => {
    const data = await request.formData();
    const enabled = data.get('enabled') === 'true';
    try {
      const res = await serverFetch(`${API_BASE}/api/v1/notifications/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        return { success: false, error: body.message ?? 'Failed to update notification settings' };
      }
      return { success: true };
    } catch {
      return { success: false, error: 'Could not reach the API' };
    }
  },
};
