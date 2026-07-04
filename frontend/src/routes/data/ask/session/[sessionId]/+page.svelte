<script lang="ts">
	import { browser } from "$app/environment";
	import { page } from "$app/state";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { getQuerySession, type QuerySessionResponse } from "$lib/api/ask";
	import { setQuerySessions, type SidebarQuerySession } from "$lib/ask/query-sessions";
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
				time: formatDate(sessionData.createdAt),
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

	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

		if (diffDays === 0) {
			return "Сегодня";
		} else if (diffDays === 1) {
			return "Вчера";
		} else if (diffDays < 7) {
			return `${diffDays} дн. назад`;
		} else {
			return date.toLocaleDateString("ru-RU", {
				day: "2-digit",
				month: "2-digit",
				year: "numeric"
			});
		}
	}

	async function goBack() {
		await goto(resolve("/data/ask"));
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-4">
		<Button variant="outline" size="icon" onclick={goBack}>
			<ArrowLeftIcon class="size-4" />
		</Button>
		<h1 class="text-2xl font-bold">Сессия запроса</h1>
	</div>

	{#if isLoading}
		<div class="grid gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(20rem,0.9fr)]">
			<div class="space-y-6">
				<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6">
					<Skeleton class="h-7 w-64 rounded-full mb-4" />
					<Skeleton class="h-4 w-full rounded-full mb-2" />
					<Skeleton class="h-4 w-3/4 rounded-full" />
				</div>
				<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6">
					<Skeleton class="h-7 w-64 rounded-full mb-4" />
					<Skeleton class="h-32 w-full rounded-2xl" />
				</div>
			</div>
			<div class="bg-card/90 rounded-[1.75rem] border border-border/60 p-6">
				<Skeleton class="h-80 w-full rounded-2xl" />
			</div>
		</div>
	{:else if errorMessage}
		<div class="text-destructive bg-destructive/10 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">
			{errorMessage}
		</div>
	{:else if session}
		<div class="grid gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(20rem,0.9fr)]">
			<div class="space-y-6">
				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-4 border-b border-border/60 pb-5">
						<div class="flex flex-wrap items-center gap-2">
							<Card.Title class="text-lg">Запрос</Card.Title>
						</div>
						<Card.Description>Оригинальный запрос пользователя к базе знаний.</Card.Description>
					</Card.Header>
					<Card.Content class="pt-5">
						<p class="whitespace-pre-wrap break-words">{session.query}</p>
					</Card.Content>
				</Card.Root>

				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-2 border-b border-border/60 pb-5">
						<Card.Title class="text-lg">Ответ</Card.Title>
						<Card.Description>Ответ от системы поиска по базе знаний.</Card.Description>
					</Card.Header>
					<Card.Content class="space-y-4 pt-5">
						<div class="bg-background/80 max-h-[32rem] overflow-auto rounded-2xl border border-border/60 p-4 text-sm leading-6 whitespace-pre-wrap break-words">
							{session.answer}
						</div>
					</Card.Content>
				</Card.Root>
			</div>

			<div class="space-y-6">
				<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
					<Card.Header class="gap-2 border-b border-border/60 pb-5">
						<Card.Title class="text-lg">Информация о сессии</Card.Title>
						<Card.Description>Дополнительные данные о запросе.</Card.Description>
					</Card.Header>
					<Card.Content class="space-y-5 pt-5">
						<div>
							<p class="text-muted-foreground mb-2 text-sm">Время</p>
							<p class="text-sm">{session.time}</p>
						</div>

						<div>
							<p class="text-muted-foreground mb-2 text-sm">Режим</p>
							<p class="text-sm">{session.mode || "naive"}</p>
						</div>

						<div>
							<p class="text-muted-foreground mb-2 text-sm">ID сессии</p>
							<p class="text-sm font-mono break-all">{session.id}</p>
						</div>
					</Card.Content>
				</Card.Root>
			</div>
		</div>
	{/if}
</div>