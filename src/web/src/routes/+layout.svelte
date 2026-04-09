<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { energyStream } from '$lib/stores/energy';

  let darkMode = false;

  onMount(() => {
    darkMode = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const stopStream = energyStream.start();
    return stopStream;
  });

  function toggleDark() {
    darkMode = !darkMode;
  }
</script>

<div class:dark={darkMode} class="min-h-screen bg-white text-gray-900 dark:bg-gray-950 dark:text-gray-100 transition-colors">
  <header class="border-b border-gray-200 dark:border-gray-800 px-6 py-4 flex items-center justify-between">
    <h1 class="text-xl font-semibold tracking-tight">Smart Charging</h1>
    <button
      on:click={toggleDark}
      aria-label="Toggle dark mode"
      class="rounded-md p-2 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
    >
      {#if darkMode}
        ☀️
      {:else}
        🌙
      {/if}
    </button>
  </header>

  <main class="px-6 py-8 max-w-5xl mx-auto">
    <slot />
  </main>
</div>
