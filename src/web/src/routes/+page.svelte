<script lang="ts">
  import { enhance } from '$app/forms';
  import { invalidateAll } from '$app/navigation';
  import type { PageData, ActionData } from './$types';
  import { energyStream } from '$lib/stores/energy';
  import CurrentReading from '$lib/components/CurrentReading.svelte';
  import PowerChart from '$lib/components/PowerChart.svelte';

  let { data, form }: { data: PageData; form: ActionData } = $props();

  const current = $derived($energyStream ?? data.currentReading ?? null);
  const history = $derived(data.history ?? []);

  let sending = $state(false);
  let toggling = $state(false);
  let toggleError = $state<string | null>(null);
  let notificationsEnabled = $state(false);

  $effect(() => {
    notificationsEnabled = data.notificationsEnabled ?? false;
  });
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
    <div class="rounded-xl border border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 p-6 flex flex-col gap-6">

      {#if data.notificationsEnabled !== null}
        <div class="flex items-center justify-between gap-4">
          <div>
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300">Enable notifications</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Receive Pushover alerts when excess solar power is available.
            </p>
          </div>
          <form
            method="POST"
            action="?/toggleNotifications"
            use:enhance={() => {
              toggling = true;
              toggleError = null;
              const newEnabled = !notificationsEnabled;
              return async ({ result }) => {
                if (result.type === 'success' && (result.data as { success: boolean } | undefined)?.success === true) {
                  notificationsEnabled = newEnabled;
                  await invalidateAll();
                } else {
                  toggleError =
                    (result.data as { error?: string } | undefined)?.error ??
                    'Failed to update notification settings.';
                }
                toggling = false;
              };
            }}
          >
            <input type="hidden" name="enabled" value={String(!notificationsEnabled)} />
            <button
              type="submit"
              role="switch"
              aria-checked={notificationsEnabled}
              aria-label="Toggle notifications"
              disabled={toggling}
              class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed dark:focus:ring-offset-gray-950 {notificationsEnabled
                ? 'bg-blue-600'
                : 'bg-gray-300 dark:bg-gray-700'}"
            >
              <span
                class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {notificationsEnabled
                  ? 'translate-x-5'
                  : 'translate-x-0'}"
              ></span>
            </button>
          </form>
        </div>

        {#if toggleError}
          <p role="alert" class="text-sm text-red-600 dark:text-red-400">{toggleError}</p>
        {/if}
      {:else}
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Notifications are not configured. Set <code class="font-mono text-xs">PUSHOVER_API_TOKEN</code> and <code class="font-mono text-xs">PUSHOVER_USER_KEY</code> to enable them.
        </p>
      {/if}

      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between pt-4 border-t border-gray-200 dark:border-gray-800">
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
            disabled={sending || !notificationsEnabled || data.notificationsEnabled === null}
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
        <p role="status" class="text-sm text-green-600 dark:text-green-400">
          Notification sent successfully.
        </p>
      {:else if form?.success === false}
        <p role="alert" class="text-sm text-red-600 dark:text-red-400">
          {form.error ?? 'Failed to send notification.'}
        </p>
      {/if}
    </div>
  </section>
</div>
