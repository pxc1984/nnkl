<script lang="ts">
	import {prependQuerySession} from "$lib/ask/query-sessions";
	import {Button} from "$lib/components/ui/button/index.js";
	import MarkdownRenderer from "$lib/components/markdown-renderer.svelte";
	import {ArrowUpIcon, GlobeIcon, FileTextIcon} from "@lucide/svelte";
	import {askQuestion, type AskResponse, type Reference} from "$lib/api/ask";
	import {getApiErrorMessage} from "$lib/api/auth";
	import {goto} from "$app/navigation";

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
			answer = await askQuestion(query, mode);

			if (answer.sessionId) {
				prependQuerySession({
					id: answer.sessionId,
					query,
					answer: answer.answer,
				});
			}
		} catch (error) {
			answer = null;
			errorMessage = getApiErrorMessage(error, "Не удалось получить ответ от базы знаний.");
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
	
	// Function to validate document ID format before creating links
	function isValidDocumentId(id: string): boolean {
		// Check if it's a proper UUID format (with or without doc- prefix)
		const uuidPattern = /^(doc-)?[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;
		// Or check if it's a proper SHA256 format (with or without doc- prefix)
		const sha256Pattern = /^(doc-)?[a-f0-9]{64}$/i;
		
		return uuidPattern.test(id) || sha256Pattern.test(id);
	}
	
	// Extract document links from references. Supports enriched backend format
	// ({ id, filename, type, createdAt }) and legacy LightRAG shapes.
	function getDocumentLinks(): Array<Reference & { link: string }> {
		if (!answer?.references) return [];

		let refsArray: any[] = [];
		if (typeof answer.references === 'string') {
			try {
				const parsed = JSON.parse(answer.references);
				refsArray = Array.isArray(parsed) ? parsed : [parsed];
			} catch {
				return [];
			}
		} else if (Array.isArray(answer.references)) {
			refsArray = answer.references;
		} else if (typeof answer.references === 'object') {
			refsArray = [answer.references];
		}

		const uuidRe = /[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}/i;
		const seen = new Set<string>();
		const result: Array<Reference & { link: string }> = [];

		for (const ref of refsArray) {
			if (!ref || typeof ref !== 'object') continue;

			let id = '';
			let filename = '';
			let type = '';
			let createdAt = '';

			if (typeof ref.id === 'string' && isValidDocumentId(ref.id)) {
				// Enriched format returned by backend.
				id = ref.id.toLowerCase();
				filename = typeof ref.filename === 'string' ? ref.filename : '';
				type = typeof ref.type === 'string' ? ref.type : '';
				createdAt = typeof ref.createdAt === 'string' ? ref.createdAt : '';
			} else {
				// Fallback for legacy LightRAG-shaped references.
				for (const key of ['file_path', 'source_id', 'reference_id', 'document_id', 'id']) {
					const value = ref[key];
					if (typeof value !== 'string') continue;
					const match = value.match(uuidRe);
					if (match && isValidDocumentId(match[0])) {
						id = match[0].toLowerCase();
						break;
					}
				}
			}

			if (!id || seen.has(id)) continue;
			seen.add(id);
			result.push({ id, filename, type, createdAt, link: `/data/${id}` });
		}

		return result;
	}
	
	// Navigate to document page
	function goToDocument(id: string) {
		// Validate the ID before navigating
		if (isValidDocumentId(id)) {
			void goto(`/data/${id}`);
		} else {
			console.warn(`Invalid document ID: ${id}`);
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
					
					<!-- Display references as clickable links -->
					{#if getDocumentLinks().length > 0}
						<div class="mt-6 pt-4 border-t border-border/60">
							<h3 class="text-sm font-medium text-foreground mb-3">Источники:</h3>
							<div class="flex flex-wrap gap-2">
								{#each getDocumentLinks() as ref}
									<button
										type="button"
										onclick={() => goToDocument(ref.id)}
										class="inline-flex max-w-full items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary transition-colors hover:bg-primary/20"
										title={ref.filename || ref.id}
									>
										<FileTextIcon class="size-3.5 shrink-0" />
										<span class="truncate">{ref.filename || `Документ ${ref.id.substring(0, 8)}...`}</span>
										{#if ref.type}
											<span class="shrink-0 opacity-70">· {ref.type}</span>
										{/if}
									</button>
								{/each}
							</div>
						</div>
					{/if}
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

		<div class="sticky bottom-0 pt-6">
			<div class="border-border/60 bg-background/80 rounded-[1.5rem] border px-4 py-4 md:px-5">
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
	</section>
</main>