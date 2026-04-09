<script lang="ts">
  import type { EnergyReading } from '$lib/types';

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
  const HEIGHT = 220;
  const PAD = { top: 10, right: 20, bottom: 45, left: 65 };

  function niceInterval(range: number, target: number): number {
    const rough = range / target;
    const mag = Math.pow(10, Math.floor(Math.log10(rough)));
    const n = rough / mag;
    const nice = n < 1.5 ? 1 : n < 3.5 ? 2 : n < 7.5 ? 5 : 10;
    return nice * mag;
  }

  function formatTime(ts: number): string {
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function formatWatts(w: number): string {
    if (Math.abs(w) >= 1000) return `${(w / 1000).toFixed(1)} kW`;
    return `${w} W`;
  }

  $: chartData = (() => {
    if (points.length === 0) return null;
    const xs = points.map((r) => new Date(r.time).getTime());
    const ys = points.map((r) => r.power_w);
    const minX = xs[0];
    const maxX = xs[xs.length - 1];
    const rawMinY = Math.min(...ys, 0);
    const rawMaxY = Math.max(...ys, 0);
    const rawRange = rawMaxY - rawMinY || 100;
    const minY = rawMinY - rawRange * 0.05;
    const maxY = rawMaxY + rawRange * 0.05;
    const rangeX = maxX - minX || 1;
    const rangeY = maxY - minY;
    const plotW = WIDTH - PAD.left - PAD.right;
    const plotH = HEIGHT - PAD.top - PAD.bottom;

    const cx = (x: number) => PAD.left + ((x - minX) / rangeX) * plotW;
    const cy = (y: number) => PAD.top + plotH - ((y - minY) / rangeY) * plotH;

    const yInterval = niceInterval(rawRange, 5);
    const yTickStart = Math.ceil(rawMinY / yInterval) * yInterval;
    const yTicks: number[] = [];
    for (let v = yTickStart; v <= rawMaxY + 0.001; v += yInterval) {
      yTicks.push(Math.round(v));
    }

    const xTicks = Array.from({ length: 7 }, (_, i) => minX + (i / 6) * rangeX);

    const path = points
      .map((r, i) => {
        const x = cx(xs[i]);
        const y = cy(r.power_w);
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');

    return { cx, cy, yTicks, xTicks, path, zeroY: cy(0) };
  })();
</script>

<div class="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700 p-4">
  {#if points.length === 0}
    <p class="text-gray-400 text-sm text-center py-8">No data yet — waiting for first reading…</p>
  {:else if chartData}
    <svg
      viewBox="0 0 {WIDTH} {HEIGHT}"
      class="w-full"
      role="img"
      aria-label="Power usage over the last 24 hours"
    >
      <!-- Y-axis grid lines and labels -->
      {#each chartData.yTicks as tick}
        <line
          x1={PAD.left}
          y1={chartData.cy(tick)}
          x2={WIDTH - PAD.right}
          y2={chartData.cy(tick)}
          class="stroke-gray-100 dark:stroke-gray-800"
          stroke-width="1"
        />
        <text
          x={PAD.left - 6}
          y={chartData.cy(tick)}
          text-anchor="end"
          dominant-baseline="middle"
          class="fill-gray-400 dark:fill-gray-500"
          font-size="11"
        >{formatWatts(tick)}</text>
      {/each}

      <!-- Zero line -->
      <line
        x1={PAD.left}
        y1={chartData.zeroY}
        x2={WIDTH - PAD.right}
        y2={chartData.zeroY}
        class="stroke-gray-300 dark:stroke-gray-600"
        stroke-width="1"
        stroke-dasharray="4 4"
      />

      <!-- X-axis tick marks and time labels -->
      {#each chartData.xTicks as tick}
        <line
          x1={chartData.cx(tick)}
          y1={HEIGHT - PAD.bottom}
          x2={chartData.cx(tick)}
          y2={HEIGHT - PAD.bottom + 4}
          class="stroke-gray-300 dark:stroke-gray-600"
          stroke-width="1"
        />
        <text
          x={chartData.cx(tick)}
          y={HEIGHT - PAD.bottom + 16}
          text-anchor="middle"
          class="fill-gray-400 dark:fill-gray-500"
          font-size="11"
        >{formatTime(tick)}</text>
      {/each}

      <!-- Y-axis title -->
      <text
        x={12}
        y={HEIGHT / 2}
        text-anchor="middle"
        transform="rotate(-90, 12, {HEIGHT / 2})"
        class="fill-gray-500 dark:fill-gray-400"
        font-size="11"
      >Power</text>

      <!-- X-axis title -->
      <text
        x={PAD.left + (WIDTH - PAD.left - PAD.right) / 2}
        y={HEIGHT - 4}
        text-anchor="middle"
        class="fill-gray-500 dark:fill-gray-400"
        font-size="11"
      >Time</text>

      <!-- Data line -->
      <path
        d={chartData.path}
        fill="none"
        class="stroke-blue-500"
        stroke-width="1.5"
        stroke-linejoin="round"
      />
    </svg>
  {/if}
</div>
