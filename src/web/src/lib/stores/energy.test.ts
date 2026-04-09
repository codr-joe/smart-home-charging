import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fetchHistory } from '../stores/energy';

describe('fetchHistory', () => {
	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('calls fetch with the correct endpoint and query parameters', async () => {
		const readings = [{ time: '2026-04-09T10:00:00.000Z', power_w: 400 }];
		vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify(readings), { status: 200 }));

		const from = new Date('2026-04-09T08:00:00.000Z');
		const to = new Date('2026-04-09T12:00:00.000Z');
		await fetchHistory(from, to, 100);

		const url = vi.mocked(fetch).mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/energy/history');
		expect(url).toContain('limit=100');
		expect(url).toContain('from=');
		expect(url).toContain('to=');
	});

	it('returns the default limit of 500 when no limit is specified', async () => {
		const readings: never[] = [];
		vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify(readings), { status: 200 }));

		await fetchHistory(new Date(), new Date());

		const url = vi.mocked(fetch).mock.calls[0][0] as string;
		expect(url).toContain('limit=500');
	});

	it('returns parsed readings on a successful response', async () => {
		const readings = [{ time: '2026-04-09T10:00:00.000Z', power_w: 400 }];
		vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify(readings), { status: 200 }));

		const result = await fetchHistory(new Date(), new Date());

		expect(result).toEqual(readings);
	});

	it('throws when the response status is not ok', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(new Response('Server Error', { status: 500 }));

		await expect(fetchHistory(new Date(), new Date())).rejects.toThrow(
			'Failed to fetch history: 500'
		);
	});
});
