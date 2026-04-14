<script lang="ts">
  import type { EnergyReading } from '$lib/types';

  let { history = [], live = null }: { history: EnergyReading[]; live: EnergyReading | null } = $props();

  const points = $derived.by(() => {
    const reversed = [...history].reverse();
    if (live) {
      const last = reversed[reversed.length - 1];
      if (!last || last.time !== live.time) {
        return [...reversed, live].slice(-720);
      }
    }
    return reversed;
  });

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

  function formatTime(ts: number, showDate: boolean): string {
    const d = new Date(ts);
    if (showDate) {
      return d.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    }
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function formatWatts(w: number): string {
    if (Math.abs(w) >= 1000) return `${(w / 1000).toFixed(1)} kW`;
    return `${w} W`;
  }

  function excessBand(excess: number): number {
    if (excess < 1000) return 0;
    return Math.floor(excess / 500) * 500;
  }

  const chartData = $derived.by(() => {
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

    const FOUR_HOURS = 4 * 60 * 60 * 1000;
    const tickStart = Math.ceil(minX / FOUR_HOURS) * FOUR_HOURS;
    const xTicks: number[] = [];
    for (let t = tickStart; t <= maxX; t += FOUR_HOURS) {
      xTicks.push(t);
    }

    const todayDate = new Date(maxX).toDateString();
    const xTicksWithLabel = xTicks.map((t) => ({
      ts: t,
      showDate: new Date(t).toDateString() !== todayDate,
    }));
    const path = points
      .map((r, i) => {
        const x = cx(xs[i]);
        const y = cy(r.power_w);
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');

    const zeroY = cy(0);

    let lastBand = 0;
    const notificationDots: Array<{ x: number; y: number }> = [];
    for (let i = 0; i < points.length; i++) {
      const r = points[i];
      const excess = r.power_w < 0 ? -r.power_w : 0;
      const band = excessBand(excess);
      if (band > lastBand) {
        notificationDots.push({ x: cx(xs[i]), y: cy(r.power_w) });
        lastBand = band;
      } else if (excess < 500 && lastBand > 0) {
        notificationDots.push({ x: cx(xs[i]), y: cy(r.power_w) });
        lastBand = 0;
      }
    }

    const tariffLines: Array<{ x: number; tariff: string }> = [];
    let prevTariff: string | undefined;
    for (let i = 0; i < points.length; i++) {
      const t = points[i].tariff;
      if (t && t !== prevTariff) {
        if (prevTariff !== undefined) {
          tariffLines.push({ x: cx(xs[i]), tariff: t });
        }
        prevTariff = t;
      }
    }

    return { cx, cy, yTicks, xTicksWithLabel, path, zeroY, plotW, notificationDots, tariffLines };
  });
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
      <defs>
        <clipPath id="clip-grid-import">
          <rect
            x={PAD.left}
            y={PAD.top}
            width={chartData.plotW}
            height={Math.max(0, chartData.zeroY - PAD.top)}
          />
        </clipPath>
        <clipPath id="clip-solar-excess">
          <rect
            x={PAD.left}
            y={chartData.zeroY}
            width={chartData.plotW}
            height={Math.max(0, HEIGHT - PAD.bottom - chartData.zeroY)}
          />
        </clipPath>
      </defs>
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
      {#each chartData.xTicksWithLabel as tick}
        <line
          x1={chartData.cx(tick.ts)}
          y1={HEIGHT - PAD.bottom}
          x2={chartData.cx(tick.ts)}
          y2={HEIGHT - PAD.bottom + 4}
          class="stroke-gray-300 dark:stroke-gray-600"
          stroke-width="1"
        />
        <text
          x={chartData.cx(tick.ts)}
          y={HEIGHT - PAD.bottom + 16}
          text-anchor="middle"
          class="fill-gray-400 dark:fill-gray-500"
          font-size="11"
        >{formatTime(tick.ts, tick.showDate)}</text>
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

      <!-- Tariff transition lines -->
      {#each chartData.tariffLines as tl}
        <line
          x1={tl.x}
          y1={PAD.top}
          x2={tl.x}
          y2={HEIGHT - PAD.bottom}
          class="stroke-amber-500 dark:stroke-amber-400"
          stroke-width="1"
          stroke-dasharray="3 3"
        />
        <text
          x={tl.x + 3}
          y={PAD.top + 10}
          text-anchor="start"
          class="fill-amber-600 dark:fill-amber-400"
          font-size="9"
        >{tl.tariff}</text>
      {/each}

      <!-- Grid import line (positive power_w, above the zero line) -->
      <path
        d={chartData.path}
        fill="none"
        class="stroke-blue-500"
        stroke-width="1.5"
        stroke-linejoin="round"
        clip-path="url(#clip-grid-import)"
      />

      <!-- Solar excess line (negative power_w, below the zero line) -->
      <path
        d={chartData.path}
        fill="none"
        class="stroke-green-500"
        stroke-width="1.5"
        stroke-linejoin="round"
        clip-path="url(#clip-solar-excess)"
      />

      <!-- Notification dots -->
      {#each chartData.notificationDots as dot}
        <circle cx={dot.x} cy={dot.y} r="4" class="fill-red-500" />
      {/each}
    </svg>
  {/if}
</div>
