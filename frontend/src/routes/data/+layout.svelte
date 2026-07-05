<script lang="ts">
	import type { Snippet } from "svelte";
	import { page } from "$app/state";
	import { onMount } from "svelte";
	import { listQuerySessions } from "$lib/api/ask";
	import { setQuerySessions } from "$lib/ask/query-sessions";
	import { authState } from "$lib/auth/store";
	import AppSidebar from "$lib/components/app-sidebar.svelte";
import * as Breadcrumb from "$lib/components/ui/breadcrumb/index.js";
import * as Sidebar from "$lib/components/ui/sidebar/index.js";

	type BCItem = { label: string; href?: string };

	let breadcrumbs = $derived.by(() => {
		const path = page.url.pathname;

		if (path === "/data" || path === "/data/") {
			return [{ label: "Материалы" }] as BCItem[];
		}

		if (path === "/data/upload") {
			return [{ label: "Материалы", href: "/data" }, { label: "Загрузить" }] as BCItem[];
		}

		if (path.startsWith("/data/ask")) {
			return [{ label: "Материалы", href: "/data" }, { label: "Поиск" }] as BCItem[];
		}

		if (path.startsWith("/data/graph")) {
			return [{ label: "Материалы", href: "/data" }, { label: "Карта знаний" }] as BCItem[];
		}

		if (path === "/data/account") {
			return [{ label: "Аккаунт" }] as BCItem[];
		}

		if (path.startsWith("/data/")) {
			return [{ label: "Материалы", href: "/data" }, { label: "Документ" }] as BCItem[];
		}

		return [{ label: "Материалы" }] as BCItem[];
	});

	const isGraphRoute = $derived(page.url.pathname.startsWith("/data/graph"));
	const isAskRoute = $derived(page.url.pathname.startsWith("/data/ask"));

	let { children }: { children: Snippet } = $props();

	onMount(() => {
		void loadQuerySessions();
	});

	async function loadQuerySessions(): Promise<void> {
		try {
			const sessions = await listQuerySessions();
			setQuerySessions(sessions);
		} catch {
			setQuerySessions([]);
		}
	}
</script>

<Sidebar.Provider>
	<AppSidebar currentUser={$authState.user} />
	<Sidebar.Inset class={`min-h-svh bg-background ${isAskRoute ? "relative overflow-hidden" : ""} md:min-h-[calc(100svh-1rem)]`}>
		<header
			class={`flex h-14 shrink-0 items-center transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12 ${isAskRoute ? "absolute inset-x-0 top-0 z-20 bg-gradient-to-b from-black/75 via-black/35 to-transparent" : ""}`}
		>
			<div class="flex items-center px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						{#each breadcrumbs as item, index (item.label)}
							<Breadcrumb.Item class={index < breadcrumbs.length - 1 ? "hidden md:block" : ""}>
								{#if item.href}
									<Breadcrumb.Link href={item.href}>{item.label}</Breadcrumb.Link>
								{:else}
									<Breadcrumb.Page>{item.label}</Breadcrumb.Page>
								{/if}
							</Breadcrumb.Item>
							{#if index < breadcrumbs.length - 1}
								<Breadcrumb.Separator class="hidden md:block" />
							{/if}
						{/each}
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>
		</header>
		<main class={`flex min-h-0 flex-1 flex-col ${isGraphRoute ? "px-0" : isAskRoute ? "px-0 pb-0 pt-0" : "px-6 pb-8 pt-2 md:px-10"}`}>
			{@render children()}
		</main>
	</Sidebar.Inset>
</Sidebar.Provider>
