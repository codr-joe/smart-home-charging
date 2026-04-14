<script lang="ts">
  import type { EnergyReading } from '$lib/types';

  let { reading }: { reading: EnergyReading | null } = $props();

  const excessW = $derived(reading && reading.power_w < 0 ? -reading.power_w : 0);
  const isInjecting = $derived(reading ? reading.power_w < 0 : false);
  const formattedPower = $derived(reading ? Math.abs(reading.power_w).toFixed(0) : '—');
  const formattedExcess = $derived(excessW.toFixed(0));
  const updatedAt = $derived(reading ? new Date(reading.time).toLocaleTimeString() : null);
</script>

<div class="grid grid-cols-2 gap-4 sm:grid-cols-3">
  <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-5 space-y-1">
    <p class="text-xs text-gray-500 dark:text-gray-400">Grid Power</p>
    <p class="text-3xl font-bold tabular-nums {isInjecting ? 'text-green-600 dark:text-green-400' : 'text-red-500 dark:text-red-400'}">
      {isInjecting ? '-' : ''}{formattedPower} <span class="text-base font-normal">W</span>
    </p>
    <p class="text-xs text-gray-400">{isInjecting ? 'Injecting' : 'Importing'}</p>
  </div>

  <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-5 space-y-1">
    <p class="text-xs text-gray-500 dark:text-gray-400">Excess Solar</p>
    <p class="text-3xl font-bold tabular-nums text-yellow-500 dark:text-yellow-400">
      {formattedExcess} <span class="text-base font-normal">W</span>
    </p>
    <p class="text-xs text-gray-400">Available for charging</p>
  </div>

  <div class="rounded-xl border border-gray-200 dark:border-gray-700 p-5 space-y-1">
    <p class="text-xs text-gray-500 dark:text-gray-400">Tariff</p>
    <p class="text-3xl font-bold tabular-nums">
      {reading?.tariff ?? '—'}
    </p>
    {#if updatedAt}
      <p class="text-xs text-gray-400">Updated {updatedAt}</p>
    {/if}
  </div>
</div>
