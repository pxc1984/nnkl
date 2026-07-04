<script lang="ts">
	import { browser } from "$app/environment";
	import { page } from "$app/state";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { getQuerySession, type QuerySessionResponse } from "$lib/api/ask";
	import { formatSessionTime, type SidebarQuerySession } from "$lib/ask/query-sessions";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import { ArrowLeftIcon } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";

	let session = $state<SidebarQuerySession | null>(null);
	let isLoading = $state(true);
	let errorMessage = $state("");

	$effect(() => {
		if (!browser) {
			return;
		}

		const sessionId = page.params.sessionId;
		if (!sessionId) {
			errorMessage = "Не указан ID сессии.";
			isLoading = false;
			return;
		}

		loadSession(sessionId);
	});

	async function loadSession(sessionId: string) {
		isLoading = true;
		errorMessage = "";

		try {
			const sessionData: QuerySessionResponse = await getQuerySession(sessionId);
			
			// Convert to SidebarQuerySession format
			session = {
				id: sessionData.id,
				name: sessionData.query,
				preview: sessionData.answer.substring(0, 120) + (sessionData.answer.length > 120 ? '...' : ''),
				time: formatSessionTime(sessionData.createdAt),
				query: sessionData.query,
				answer: sessionData.answer,
				mode: sessionData.mode,
				active: true
			};
		} catch (error) {
			errorMessage = getApiErrorMessage(error, "Не удалось загрузить сессию.");
		} finally {
			isLoading = false;
		}
	}

	async function goBack() {
		await goto(resolve("/data/ask"));
	}
</script>

<div class="mx-auto flex w-full max-w-3xl flex-col py-6">
	<div class="mb-10 flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={goBack}>
			<ArrowLeftIcon class="size-4" />
		</Button>

		<h1 class="text-2xl font-semibold tracking-tight">
			Сессия запроса
		</h1>
	</div>

	{#if isLoading}
		<div class="space-y-10">
			<div>
				<Skeleton class="mb-2 h-5 w-24" />
				<Skeleton class="mb-1 h-4 w-full" />
				<Skeleton class="h-4 w-3/4" />
			</div>

			<div>
				<Skeleton class="mb-4 h-5 w-24" />
				<Skeleton class="h-48 w-full" />
			</div>
		</div>
	{:else if errorMessage}
		<div class="text-sm text-destructive">
			{errorMessage}
		</div>
	{:else if session}
		<div class="space-y-12">

			<section>
				<h2 class="mb-3 text-sm font-medium text-muted-foreground">
					Запрос
				</h2>

				<p class="whitespace-pre-wrap leading-7 break-words text-foreground">
					{session.query}
				</p>
			</section>

			<section class="space-y-6">
				<div>
					{session.answer}
				</div>
			</section>

			<aside class="rounded-2xl border border-border/20 bg-muted/20 p-5">
				<h2 class="mb-4 text-sm font-medium text-muted-foreground">
					Информация
				</h2>

				<div class="space-y-4 text-sm">

					<div>
						<p class="text-xs text-muted-foreground">
							Время
						</p>

						<p class="mt-0.5">{session.time}</p>
					</div>

					<div>
						<p class="text-xs text-muted-foreground">
							Режим
						</p>

						<p class="mt-0.5">{session.mode || "naive"}</p>
					</div>

					<div>
						<p class="text-xs text-muted-foreground">
							ID сессии
						</p>

						<p class="mt-0.5 font-mono break-all text-xs">
							{session.id}
						</p>
					</div>

				</div>
			</aside>
		</div>
	{/if}
</div>
