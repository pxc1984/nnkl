<script lang="ts">
	import { Button } from "$lib/components/ui/button/index.js";
	import { ArrowUpIcon, GlobeIcon } from "@lucide/svelte";
	import { streamQuestion, type AskResponse } from "$lib/api/ask";
	import { getApiErrorMessage } from "$lib/api/auth";

	let prompt = $state("");
	let useDomesticSources = $state(false);
	let isLoading = $state(false);
	let answer = $state<AskResponse | null>(null);
	let errorMessage = $state("");

	async function handleSubmit() {
		const query = prompt.trim();
		if (!query || isLoading) {
			return;
		}

		isLoading = true;
		errorMessage = "";
		answer = null;

		try {
			const mode = useDomesticSources ? "local" : "naive";
			answer = { answer: "", mode };
			await streamQuestion(query, mode, (chunk) => {
				if (answer) {
					answer = { ...answer, answer: answer.answer + chunk };
				}
			});
		} catch (error) {
			errorMessage = getApiErrorMessage(error, "Не удалось получить ответ от базы знаний.");
		} finally {
			isLoading = false;
		}
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Enter" && (!event.metaKey && !event.ctrlKey)) {
			event.preventDefault();
			void handleSubmit();
		}
	}
</script>

<main class="flex flex-1 items-center justify-center px-4 pb-10 pt-4 md:px-8">
	<section class="flex w-full max-w-4xl flex-col items-center gap-8">
		<div class="space-y-3 text-center">
			<p class="text-muted-foreground text-sm tracking-[0.24em] uppercase">Поиск по базе знаний</p>
			<h1 class="text-foreground text-3xl font-semibold tracking-tight md:text-5xl">
				Что у вас сегодня на уме?
			</h1>
		</div>

		<div class="bg-card/90 w-full rounded-[2rem] border border-border/60 shadow-[0_24px_80px_-32px_rgba(0,0,0,0.45)] backdrop-blur">
			<div class="border-border/60 bg-background/70 flex min-h-36 flex-col rounded-[1.5rem] border px-4 py-5 md:px-6 md:py-6">
				<div class="mb-6">
					<textarea
						bind:value={prompt}
						onkeydown={handleKeyDown}
						rows="4"
						placeholder="Какие способы закачки шахтных вод в глубокие горизонты применялись в России и за рубежом, и каковы их технико-экономические показатели?"
						class="text-foreground placeholder:text-muted-foreground field-sizing-content min-h-28 w-full resize-none border-0 bg-transparent px-0 py-0 text-base leading-7 shadow-none outline-none focus-visible:border-0 focus-visible:ring-0 md:text-lg"
					></textarea>
				</div>

				<div class="mt-6 flex flex-col gap-4 md:flex-row md:items-end md:justify-between md:gap-6">
					<div class="flex flex-wrap gap-3">
						<button
							type="button"
							class={useDomesticSources
								? "from-white via-blue-500 to-red-500 text-black inline-flex items-center gap-2 rounded-full bg-linear-to-r px-3 py-2 text-sm transition-colors"
								: "bg-muted text-muted-foreground hover:bg-muted/80 inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm transition-colors"}
							onclick={() => (useDomesticSources = !useDomesticSources)}
						>
							<GlobeIcon class="size-4" />
							{useDomesticSources ? "Отечественные источники" : "Все источники"}
						</button>
					</div>

					<div class="flex items-center justify-end gap-3">
						<Button
							type="button"
							size="icon"
							class="rounded-full"
							disabled={!prompt.trim() || isLoading}
							onclick={handleSubmit}
						>
							{#if isLoading}
								<span class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent"></span>
							{:else}
								<ArrowUpIcon class="size-4" />
							{/if}
						</Button>
					</div>
				</div>
			</div>
		</div>

		{#if errorMessage}
			<div class="w-full rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
				{errorMessage}
			</div>
		{/if}

		{#if answer}
			<div class="w-full rounded-[2rem] border border-border/60 bg-card/90 p-6 shadow-[0_24px_80px_-32px_rgba(0,0,0,0.45)] backdrop-blur">
				<div class="mb-4 flex items-center justify-between">
					<p class="text-sm font-medium text-muted-foreground">Ответ базы знаний</p>
					<span class="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">{answer.mode}</span>
				</div>
				<div class="prose prose-invert max-w-none">
					{#each answer.answer.split("\n") as line, index (index)}
						<p class="mb-2 text-base leading-7">{line}</p>
					{/each}
				</div>
			</div>
		{/if}
	</section>
</main>
