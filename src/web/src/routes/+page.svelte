<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData, ActionData } from './$types';
  import { energyStream } from '$lib/stores/energy';
  import CurrentReading from '$lib/components/CurrentReading.svelte';
  import PowerChart from '$lib/components/PowerChart.svelte';

  let { data, form }: { data: PageData; form: ActionData } = $props();

  const current = $derived($energyStream ?? null);
  const history = $derived(data.history ?? []);

  let sending = $state(false);
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

  <section>
    <h2 class="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-widest mb-4">
      Notifications
    </h2>
    <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-sm text-gray-600 dark:text-gray-400">
        Send a test notification to verify your Pushover configuration is working correctly.
      </p>
      <form
        method="POST"
        action="?/testNotification"
        use:enhance={() => {
          sending = true;
          return async ({ update }) => {
            await update();
            sending = false;
          };
        }}
      >
        <button
          type="submit"
          disabled={sending}
          class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:focus:ring-offset-gray-950 transition-colors"
        >
          {#if sending}
            <span class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" aria-hidden="true"></span>
            Sending…
          {:else}
            Send Test Notification
          {/if}
        </button>
      </form>
    </div>

    {#if form?.success === true}
      <p role="status" class="mt-3 text-sm text-green-600 dark:text-green-400">
        Notification sent successfully.
      </p>
    {:else if form?.success === false}
      <p role="alert" class="mt-3 text-sm text-red-600 dark:text-red-400">
        {form.error ?? 'Failed to send notification.'}
      </p>
    {/if}
  </section>
</div>
