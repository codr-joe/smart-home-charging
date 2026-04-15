import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { load, actions } from './+page.server';

describe('page server load', () => {
	const mockHistory = [{ time: '2026-04-09T10:00:00.000Z', power_w: 500, tariff: 'T1' }];

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns history data on a successful fetch', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true }), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ time: '2026-04-09T10:00:00.000Z', power_w: 500 }), { status: 200 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual(mockHistory);
	});

	it('returns an empty history on a non-ok HTTP response', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response('', { status: 500 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true }), { status: 200 }))
			.mockResolvedValueOnce(new Response('', { status: 500 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual([]);
	});

	it('returns an empty history when the fetch call throws', async () => {
		vi.mocked(fetch).mockRejectedValueOnce(new Error('network error'));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual([]);
	});

	it('requests the last 24 hours of data with 2-minute buckets', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify([]), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true }), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify(null), { status: 200 }));

		const before = Date.now();
		await load({} as Parameters<typeof load>[0]);
		const after = Date.now();

		const url = vi.mocked(fetch).mock.calls[0][0] as string;
		const params = new URL(url).searchParams;

		const from = new Date(params.get('from')!).getTime();
		const to = new Date(params.get('to')!).getTime();

		expect(to - from).toBeCloseTo(24 * 60 * 60 * 1000, -3);
		expect(params.get('bucket')).toBe('120');
		expect(params.get('limit')).toBe('720');
		expect(to).toBeGreaterThanOrEqual(before);
		expect(to).toBeLessThanOrEqual(after + 100);
	});
});

describe('page server load — current reading', () => {
	const mockHistory = [{ time: '2026-04-09T10:00:00.000Z', power_w: 500, tariff: 'T1' }];
	const mockCurrent = { time: '2026-04-09T10:00:00.000Z', power_w: 500, tariff: 'T1' };

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns currentReading from the API on success', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true }), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify(mockCurrent), { status: 200 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.currentReading).toEqual(mockCurrent);
	});

	it('returns currentReading:null when the current API is unavailable', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ enabled: true }), { status: 200 }))
			.mockResolvedValueOnce(new Response('', { status: 500 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.currentReading).toBeNull();
	});
});

describe('testNotification action', () => {
	const mockFetch = vi.fn();

	const makeEvent = () =>
		({ fetch: mockFetch }) as unknown as Parameters<typeof actions.testNotification>[0];

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('returns success:true when the API responds with 200', async () => {
		mockFetch.mockResolvedValueOnce(new Response('{"status":"ok"}', { status: 200 }));

		const result = await actions.testNotification(makeEvent());

		expect(result).toEqual({ success: true });
	});

	it('calls POST /api/v1/notifications/test', async () => {
		mockFetch.mockResolvedValueOnce(new Response('{"status":"ok"}', { status: 200 }));

		await actions.testNotification(makeEvent());

		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/api/v1/notifications/test'),
			expect.objectContaining({ method: 'POST' })
		);
	});

	it('returns success:false with error message when API responds with non-200', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ message: 'notifications are not configured' }), { status: 503 })
		);

		const result = await actions.testNotification(makeEvent());

		expect(result).toMatchObject({ success: false, error: 'notifications are not configured' });
	});

	it('returns success:false with fallback message when API error body is not JSON', async () => {
		mockFetch.mockResolvedValueOnce(new Response('bad gateway', { status: 502 }));

		const result = await actions.testNotification(makeEvent());

		expect(result).toMatchObject({ success: false });
		expect(typeof (result as { error: string }).error).toBe('string');
	});

	it('returns success:false when fetch throws', async () => {
		mockFetch.mockRejectedValueOnce(new Error('network error'));

		const result = await actions.testNotification(makeEvent());

		expect(result).toEqual({ success: false, error: 'Could not reach the API' });
	});
});

describe('page server load — notifications settings', () => {
	const mockHistory = [{ time: '2026-04-09T10:00:00.000Z', power_w: 500, tariff: 'T1' }];

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns notificationsEnabled:true when settings API returns enabled:true', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(
				new Response(JSON.stringify({ enabled: true }), { status: 200 })
			)
			.mockResolvedValueOnce(new Response(JSON.stringify(null), { status: 200 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.notificationsEnabled).toBe(true);
	});

	it('returns notificationsEnabled:false when settings API returns enabled:false', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(
				new Response(JSON.stringify({ enabled: false }), { status: 200 })
			)
			.mockResolvedValueOnce(new Response(JSON.stringify(null), { status: 200 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.notificationsEnabled).toBe(false);
	});

	it('returns notificationsEnabled:null when settings API is unavailable', async () => {
		vi.mocked(fetch)
			.mockResolvedValueOnce(new Response(JSON.stringify(mockHistory), { status: 200 }))
			.mockResolvedValueOnce(new Response('', { status: 503 }))
			.mockResolvedValueOnce(new Response(JSON.stringify(null), { status: 200 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.notificationsEnabled).toBeNull();
	});
});

describe('toggleNotifications action', () => {
	const mockFetch = vi.fn();

	const makeEvent = (enabled: boolean) => {
		const body = new FormData();
		body.set('enabled', String(enabled));
		return {
			fetch: mockFetch,
			request: { formData: async () => body },
		} as unknown as Parameters<typeof actions.toggleNotifications>[0];
	};

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('returns success:true when API responds with 200', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ enabled: false }), { status: 200 })
		);

		const result = await actions.toggleNotifications(makeEvent(false));

		expect(result).toEqual({ success: true });
	});

	it('calls PUT /api/v1/notifications/settings with correct body', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ enabled: false }), { status: 200 })
		);

		await actions.toggleNotifications(makeEvent(false));

		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/api/v1/notifications/settings'),
			expect.objectContaining({ method: 'PUT', body: JSON.stringify({ enabled: false }) })
		);
	});

	it('returns success:false with error message when API responds with non-200', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ message: 'notifications are not configured' }), { status: 503 })
		);

		const result = await actions.toggleNotifications(makeEvent(true));

		expect(result).toMatchObject({ success: false, error: 'notifications are not configured' });
	});

	it('returns success:false when fetch throws', async () => {
		mockFetch.mockRejectedValueOnce(new Error('network error'));

		const result = await actions.toggleNotifications(makeEvent(true));

		expect(result).toEqual({ success: false, error: 'Could not reach the API' });
	});
});
