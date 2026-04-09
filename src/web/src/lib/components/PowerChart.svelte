<script lang="ts">
  import type { EnergyReading } from '$lib/types';
  import { onMount } from 'svelte';

  export let history: EnergyReading[] = [];
  export let live: EnergyReading | null = null;

  let points: EnergyReading[] = [];
  $: {
    points = [...history].reverse();
    if (live) {
      const last = points[points.length - 1];
      if (!last || last.time !== live.time) {
        points = [...points, live].slice(-288);
      }
    }
  }

  const WIDTH = 800;
  const HEIGHT = 200;
  const PAD = { top: 10, right: 10, bottom: 30, left: 50 };

  function toPath(readings: EnergyReading[]): string {
    if (readings.length === 0) return '';
    const xs = readings.map((r) => new Date(r.time).getTime());
    const ys = readings.map((r) => r.power_w);
    const minX = xs[0], maxX = xs[xs.length - 1];
    const minY = Math.min(...ys, 0);
    const maxY = Math.max(...ys, 0);
    const rangeX = maxX - minX || 1;
    const rangeY = maxY - minY || 1;
    const plotW = WIDTH - PAD.left - PAD.right;
    const plotH = HEIGHT - PAD.top - PAD.bottom;

    const cx = (x: number) => PAD.left + ((x - minX) / rangeX) * plotW;
    const cy = (y: number) => PAD.top + plotH - ((y - minY) / rangeY) * plotH;

    return readings.map((r, i) => {
      const x = cx(xs[i]);
      const y = cy(r.power_w);
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
  }

  $: path = toPath(points);
</script>

<div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700 p-4">
  {#if points.length === 0}
    <p class="text-gray-400 text-sm text-center py-8">No data yet — waiting for first reading…</p>
  {:else}
    <svg
      viewBox="0 0 {WIDTH} {HEIGHT}"
      class="w-full"
      role="img"
      aria-label="Power usage over the last 24 hours"
    >
      <!-- Zero line -->
      <line
        x1={PAD.left} y1={HEIGHT / 2}
        x2={WIDTH - PAD.right} y2={HEIGHT / 2}
        class="stroke-gray-200 dark:stroke-gray-700"
        stroke-width="1"
        stroke-dasharray="4 4"
      />
      <!-- Data line -->
      <path d={path} fill="none" class="stroke-blue-500" stroke-width="1.5" stroke-linejoin="round" />
    </svg>
  {/if}
</div>
