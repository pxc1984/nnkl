<script lang="ts">
	import { browser } from "$app/environment";
	import { page } from "$app/state";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { getQuerySession, type QuerySessionResponse } from "$lib/api/ask";
	import {
		formatSessionTime,
		setQuerySessions,
		type SidebarQuerySession,
	} from "$lib/ask/query-sessions";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import { ArrowLeftIcon, FileTextIcon } from "@lucide/svelte";
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

<div class="mx-auto flex w-full max-w-5xl flex-col">
	<div class="mb-10 flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={goBack}>
			<ArrowLeftIcon class="size-4" />
		</Button>

		<h1 class="text-2xl font-semibold tracking-tight">
			Сессия запроса
		</h1>
	</div>

	{#if isLoading}
		<div class="grid gap-12 xl:grid-cols-[minmax(0,1fr)_18rem]">
			<div class="space-y-10">
				<div>
					<Skeleton class="mb-4 h-6 w-32 rounded-full" />
					<Skeleton class="mb-2 h-4 w-full rounded-full" />
					<Skeleton class="h-4 w-3/4 rounded-full" />
				</div>

				<div>
					<Skeleton class="mb-4 h-6 w-32 rounded-full" />
					<Skeleton class="h-48 w-full rounded-2xl" />
				</div>
			</div>

			<div>
				<Skeleton class="h-72 w-full rounded-2xl" />
			</div>
		</div>
	{:else if errorMessage}
		<div class="rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
			{errorMessage}
		</div>
	{:else if session}
		<div class="grid gap-12 xl:grid-cols-[minmax(0,1fr)_18rem]">
			<div class="space-y-12">

				<section>
					<h2 class="mb-4 text-lg font-medium">
						Запрос
					</h2>

					<p class="whitespace-pre-wrap leading-7 break-words text-foreground">
						{session.query}
					</p>
				</section>

				<section>
					<h2 class="mb-6 text-lg font-medium">
						Ответ
					</h2>

					<div class="prose prose-neutral dark:prose-invert max-w-none whitespace-pre-wrap leading-7 break-words">
						{session.answer}
					</div>
				</section>

			</div>

			<aside class="h-fit rounded-2xl border border-border/20 bg-muted/20 p-5">
				<h2 class="mb-6 text-base font-medium">
					Информация
				</h2>

				<div class="space-y-6 text-sm">

					<div>
						<p class="mb-1 text-muted-foreground">
							Время
						</p>

						<p>{session.time}</p>
					</div>

					<div>
						<p class="mb-1 text-muted-foreground">
							Режим
						</p>

						<p>{session.mode || "naive"}</p>
					</div>

					<div>
						<p class="mb-1 text-muted-foreground">
							ID сессии
						</p>

						<p class="font-mono break-all text-xs">
							{session.id}
						</p>
					</div>

				</div>
			</aside>
		</div>
	{/if}
</div>
