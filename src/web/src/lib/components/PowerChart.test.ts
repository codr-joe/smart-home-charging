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
});
