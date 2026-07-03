<script lang="ts">
  import "../app.css";
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";

	import { ensureAuthenticated, hydrateAuthState } from "$lib/auth/store";

	let { children } = $props();

	let checkingAuth = $state(false);
	let guardRun = 0;

	function isAuthRoute(pathname: string): boolean {
		return pathname.startsWith("/auth");
	}

	async function guardRoute(pathname: string): Promise<void> {
		if (!browser) {
			return;
		}

		if (isAuthRoute(pathname)) {
			hydrateAuthState();
			checkingAuth = false;
			return;
		}

		const currentRun = ++guardRun;
		checkingAuth = true;

		const isAuthenticated = await ensureAuthenticated();
		if (currentRun != guardRun) {
			return;
		}

		checkingAuth = false;
		if (!isAuthenticated) {
			await goto(resolve("/auth/login"));
		}
	}

	$effect(() => {
		void guardRoute(page.url.pathname);
	});
</script>

{#if checkingAuth && !isAuthRoute(page.url.pathname)}
	<div class="bg-background min-h-screen"></div>
{:else}
	{@render children?.()}
{/if}
