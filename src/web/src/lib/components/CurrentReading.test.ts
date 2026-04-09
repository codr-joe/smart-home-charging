import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import CurrentReading from './CurrentReading.svelte';
import type { EnergyReading } from '$lib/types';

describe('CurrentReading', () => {
  it('shows dashes when no reading is provided', () => {
    render(CurrentReading, { props: { reading: null } });
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
  });

  it('shows importing label when power_w is positive', () => {
    const reading: EnergyReading = { time: new Date().toISOString(), power_w: 800, tariff: 'T1' };
    render(CurrentReading, { props: { reading } });
    expect(screen.getByText('Importing')).toBeTruthy();
  });

  it('shows injecting label and excess when power_w is negative', () => {
    const reading: EnergyReading = { time: new Date().toISOString(), power_w: -1500, tariff: 'T2' };
    render(CurrentReading, { props: { reading } });
    expect(screen.getByText('Injecting')).toBeTruthy();
    expect(screen.getByText('1500')).toBeTruthy();
  });
});
