import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { load } from './+page.server';

describe('page server load', () => {
	const mockHistory = [{ time: '2026-04-09T10:00:00.000Z', power_w: 500, tariff: 'T1' }];

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('returns history data on a successful fetch', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify(mockHistory), { status: 200 })
		);

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual(mockHistory);
	});

	it('returns an empty history on a non-ok HTTP response', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(new Response('', { status: 500 }));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual([]);
	});

	it('returns an empty history when the fetch call throws', async () => {
		vi.mocked(fetch).mockRejectedValueOnce(new Error('network error'));

		const result = await load({} as Parameters<typeof load>[0]);

		expect(result.history).toEqual([]);
	});

	it('requests the last 24 hours of data', async () => {
		vi.mocked(fetch).mockResolvedValueOnce(
			new Response(JSON.stringify([]), { status: 200 })
		);

		const before = Date.now();
		await load({} as Parameters<typeof load>[0]);
		const after = Date.now();

		const url = vi.mocked(fetch).mock.calls[0][0] as string;
		const params = new URL(url).searchParams;

		const from = new Date(params.get('from')!).getTime();
		const to = new Date(params.get('to')!).getTime();

		expect(to - from).toBeCloseTo(24 * 60 * 60 * 1000, -3);
		expect(to).toBeGreaterThanOrEqual(before);
		expect(to).toBeLessThanOrEqual(after + 100);
	});
});
