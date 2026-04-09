<script lang="ts">
  import type { PageData } from './$types';
  import { energyStream } from '$lib/stores/energy';
  import CurrentReading from '$lib/components/CurrentReading.svelte';
  import PowerChart from '$lib/components/PowerChart.svelte';

  export let data: PageData;

  $: current = $energyStream ?? null;
  $: history = data.history ?? [];
</script>

<svelte:head>
  <title>Smart Charging — Dashboard</title>
</svelte:head>

<div class="space-y-8">
  <section>
    <h2 class="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-widest mb-4">
      Current
    </h2>
    <CurrentReading reading={current} />
  </section>

  <section>
    <h2 class="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-widest mb-4">
      Last 24 Hours
    </h2>
    <PowerChart {history} live={current} />
  </section>
</div>
