import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import PowerChart from './PowerChart.svelte';
import type { EnergyReading } from '$lib/types';

describe('PowerChart', () => {
	it('shows empty state when no history and no live reading', () => {
		render(PowerChart, { props: { history: [], live: null } });
		expect(screen.getByText(/No data yet/)).toBeTruthy();
	});

	it('renders SVG chart when history is provided', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 500 },
			{ time: new Date().toISOString(), power_w: 800 }
		];
		render(PowerChart, { props: { history, live: null } });
		expect(document.querySelector('svg')).not.toBeNull();
	});

	it('renders SVG chart when only a live reading is provided', () => {
		const live: EnergyReading = { time: new Date().toISOString(), power_w: 300 };
		render(PowerChart, { props: { history: [], live } });
		expect(document.querySelector('svg')).not.toBeNull();
	});

	it('does not duplicate live reading already present in history', () => {
		const t = new Date().toISOString();
		const history: EnergyReading[] = [{ time: t, power_w: 500 }];
		const live: EnergyReading = { time: t, power_w: 500 };
		render(PowerChart, { props: { history, live } });
		expect(document.querySelector('svg')).not.toBeNull();
		expect(screen.queryByText(/No data yet/)).toBeNull();
	});

	it('includes live reading when it differs from the last history entry', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 500 }
		];
		const live: EnergyReading = { time: new Date().toISOString(), power_w: 700 };
		render(PowerChart, { props: { history, live } });
		expect(document.querySelector('svg')).not.toBeNull();
	});

	it('renders two colored path segments for dual-color line', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 500 },
			{ time: new Date().toISOString(), power_w: 800 }
		];
		render(PowerChart, { props: { history, live: null } });
		const paths = document.querySelectorAll('path');
		const strokeClasses = Array.from(paths).map((p) => p.getAttribute('class') ?? '');
		expect(strokeClasses.some((c) => c.includes('stroke-blue-500'))).toBe(true);
		expect(strokeClasses.some((c) => c.includes('stroke-green-500'))).toBe(true);
	});

	it('renders notification dot when solar excess crosses 1000 W threshold', () => {
		// History is newest-first (as returned by the API)
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: -1200 },
			{ time: new Date(Date.now() - 120000).toISOString(), power_w: 200 }
		];
		render(PowerChart, { props: { history, live: null } });
		const dots = document.querySelectorAll('circle.fill-red-500');
		expect(dots.length).toBeGreaterThanOrEqual(1);
	});

	it('renders no notification dots when solar excess stays below 1000 W', () => {
		// History is newest-first (as returned by the API)
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: -800 },
			{ time: new Date(Date.now() - 120000).toISOString(), power_w: -500 }
		];
		render(PowerChart, { props: { history, live: null } });
		const dots = document.querySelectorAll('circle.fill-red-500');
		expect(dots.length).toBe(0);
	});

	it('renders a falling notification dot when excess drops below 500 W after being above 1000 W', () => {
		// History is newest-first (as returned by the API)
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 300 },
			{ time: new Date(Date.now() - 180000).toISOString(), power_w: -1200 }
		];
		render(PowerChart, { props: { history, live: null } });
		const dots = document.querySelectorAll('circle.fill-red-500');
		expect(dots.length).toBe(2);
	});

	it('renders tariff transition lines when tariff changes in history', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 500, tariff: 'T2' },
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 400, tariff: 'T1' }
		];
		render(PowerChart, { props: { history, live: null } });
		const lines = Array.from(document.querySelectorAll('line')).filter(
			(l) => l.getAttribute('stroke-dasharray') === '3 3'
		);
		expect(lines.length).toBeGreaterThanOrEqual(1);
	});

	it('renders tariff label text when tariff transition occurs', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 500, tariff: 'T2' },
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 400, tariff: 'T1' }
		];
		render(PowerChart, { props: { history, live: null } });
		expect(screen.getByText('T2')).toBeTruthy();
	});

	it('renders no tariff lines when all readings share the same tariff', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 500, tariff: 'T1' },
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 400, tariff: 'T1' }
		];
		render(PowerChart, { props: { history, live: null } });
		const lines = Array.from(document.querySelectorAll('line')).filter(
			(l) => l.getAttribute('stroke-dasharray') === '3 3'
		);
		expect(lines.length).toBe(0);
	});

	it('renders no tariff lines when readings have no tariff data', () => {
		const history: EnergyReading[] = [
			{ time: new Date(Date.now() - 60000).toISOString(), power_w: 500 },
			{ time: new Date(Date.now() - 3600000).toISOString(), power_w: 400 }
		];
		render(PowerChart, { props: { history, live: null } });
		const lines = Array.from(document.querySelectorAll('line')).filter(
			(l) => l.getAttribute('stroke-dasharray') === '3 3'
		);
		expect(lines.length).toBe(0);
	});
});
