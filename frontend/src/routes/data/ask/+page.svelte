<script lang="ts">
	import {prependQuerySession} from "$lib/ask/query-sessions";
	import {Button} from "$lib/components/ui/button/index.js";
	import MarkdownRenderer from "$lib/components/markdown-renderer.svelte";
	import {ArrowUpIcon, GlobeIcon} from "@lucide/svelte";
	import {askQuestion, type AskResponse} from "$lib/api/ask";
	import {getApiErrorMessage} from "$lib/api/auth";

    let prompt = $state("");
    let useDomesticSources = $state(false);
    let isLoading = $state(false);
    let inlineMessage = $state("");

    const activeSession = $derived($querySessions.find((session) => session.active) ?? null);
    const answer = $derived.by<AskResponse | null>(() => {
        if (!activeSession) {
            return null;
        }

        return {
            answer: activeSession.answer,
            mode: activeSession.mode ?? "naive",
            sessionId: activeSession.id,
        };
    });

    async function handleSubmit() {
        const query = prompt.trim();
        if (!query || isLoading) {
            return;
        }

        isLoading = true;
        inlineMessage = "";

        try {
            const mode = useDomesticSources ? "local" : "naive";
            const nextAnswer = await askQuestion(query, mode);

            if (isNoContextAnswer(nextAnswer.answer)) {
                inlineMessage = "Не удалось подобрать контекст для ответа. Попробуйте переформулировать вопрос и спросить снова.";
                return;
            }

            if (nextAnswer.sessionId) {
                prependQuerySession({
                    id: nextAnswer.sessionId,
                    query,
                    answer: nextAnswer.answer,
                    mode: nextAnswer.mode,
                });
            }
        } catch {
            inlineMessage = "Не удалось получить ответ. Попробуйте спросить снова.";
        } finally {
            isLoading = false;
        }
    }

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === "Enter" && !event.shiftKey && !event.metaKey && !event.ctrlKey) {
			event.preventDefault();
			void handleSubmit();
		}
	}
</script>

<main class="flex flex-1 px-4 pb-6 pt-2 md:px-8 md:pb-8">
	<section class="mx-auto flex w-full max-w-5xl flex-1 flex-col">
		<div class="flex flex-1 flex-col gap-6">
			{#if answer}
				<div class="bg-card/90 flex-1 rounded-[2rem] border border-border/60 px-5 py-6 shadow-[0_24px_80px_-32px_rgba(0,0,0,0.45)] backdrop-blur md:px-8 md:py-8">
					<div class="mb-6 flex items-center justify-between gap-3 border-b border-border/60 pb-4">
						<div>
							<p class="text-sm font-medium text-foreground">Ответ базы знаний</p>
							<p class="text-muted-foreground text-sm">Одноразовый запрос с сохранением сессии</p>
						</div>
						<span class="rounded-full border border-border/60 bg-muted px-2.5 py-1 text-xs text-muted-foreground">{answer.mode}</span>
					</div>
					<MarkdownRenderer markdown={answer.answer} />
				</div>
			{:else}
				<div class="flex flex-1 items-center justify-center rounded-[2rem] px-6 py-16 text-center shadow-[0_24px_80px_-32px_rgba(0,0,0,0.35)] backdrop-blur">
					<div class="max-w-2xl space-y-4">
						<p class="text-muted-foreground text-sm tracking-[0.24em] uppercase">Поиск по базе знаний</p>
						<h1 class="text-foreground text-3xl font-semibold tracking-tight md:text-5xl">
							Что у вас сегодня на уме?
						</h1>
					</div>
				</div>
			{/if}

			{#if errorMessage}
				<div class="w-full rounded-2xl border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive">
					{errorMessage}
				</div>
			{/if}
		</div>

        <div class="absolute inset-x-0 bottom-0 pt-6">
            <div class="border-border/60 bg-background/95 rounded-[1.5rem] border px-4 py-4 shadow-[0_-16px_48px_-36px_rgba(0,0,0,0.8)] backdrop-blur md:px-5">
				<textarea
                        bind:value={prompt}
                        onkeydown={handleKeyDown}
                        rows="3"
                        placeholder="Какие способы закачки шахтных вод в глубокие горизонты применялись в России и за рубежом, и каковы их технико-экономические показатели?"
                        class="text-foreground placeholder:text-muted-foreground field-sizing-content min-h-24 w-full resize-none border-0 bg-transparent px-0 py-0 text-base leading-7 shadow-none outline-none focus-visible:border-0 focus-visible:ring-0 md:text-lg"
                ></textarea>

                <div class="mt-4 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
                    <div class="flex flex-wrap gap-3">
                        <button
                                type="button"
                                class={useDomesticSources
								? "from-white via-blue-500 to-red-500 text-black inline-flex items-center gap-2 rounded-full bg-linear-to-r px-3 py-2 text-sm transition-colors"
								: "bg-muted text-muted-foreground hover:bg-muted/80 inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm transition-colors"}
                                onclick={() => (useDomesticSources = !useDomesticSources)}
                        >
                            <GlobeIcon class="size-4"/>
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
                                <ArrowUpIcon class="size-4"/>
                            {/if}
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    </section>
</main>