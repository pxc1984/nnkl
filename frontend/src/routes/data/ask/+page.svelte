<script lang="ts">
	import {prependQuerySession} from "$lib/ask/query-sessions";
	import {Button} from "$lib/components/ui/button/index.js";
	import MarkdownRenderer from "$lib/components/markdown-renderer.svelte";
	import {ArrowUpIcon, GlobeIcon, FileTextIcon} from "@lucide/svelte";
	import {askQuestion, type AskResponse, type Reference} from "$lib/api/ask";
	import {getApiErrorMessage} from "$lib/api/auth";
	import {goto} from "$app/navigation";
	import {resolve} from "$app/paths";
	import {SvelteSet} from "svelte/reactivity";

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

		let refsArray: Reference[] = [];
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
		const seen = new SvelteSet<string>();
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
				const legacyRef = ref as Record<string, unknown>;
				for (const key of ['file_path', 'source_id', 'reference_id', 'document_id', 'id']) {
					const value = legacyRef[key];
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
			void goto(resolve(`/data/${id}`));
		} else {
			console.warn(`Invalid document ID: ${id}`);
		}
	}
</script>

<main class="flex flex-1 px-4 py-6">
	<section class="mx-auto flex w-full max-w-3xl flex-1 flex-col">
		<div class="flex flex-1 flex-col">
			{#if answer}
				<div class="flex-1 px-2 py-10 md:px-4">
					<MarkdownRenderer markdown={answer.answer} />

					{#if getDocumentLinks().length > 0}
						<div class="mt-10 pt-6">
							<h3 class="mb-3 text-sm font-medium text-muted-foreground">
								Источники
							</h3>

							<div class="flex flex-wrap gap-2">
								{#each getDocumentLinks() as ref (ref.id)}
									<button
											type="button"
											onclick={() => goToDocument(ref.id)}
											class="inline-flex items-center gap-2 rounded-full border border-border/15 bg-muted/20 px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted/40"
									>
										<FileTextIcon class="size-3.5" />
										<span class="truncate">
											{ref.number ? `[${ref.number}] ` : ""}{ref.filename || `Документ ${ref.id.substring(0, 8)}...`}
										</span>

										{#if ref.type}
											<span class="opacity-60">
												· {ref.type}
											</span>
										{/if}
									</button>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			{:else}
				<div class="flex flex-1 items-center justify-center py-20">
					<div class="space-y-3 text-center">
						<p class="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">
							Поиск по базе знаний
						</p>

						<h1 class="text-4xl font-medium tracking-tight text-foreground md:text-5xl">
							Что у вас сегодня на уме?
						</h1>
					</div>
				</div>
			{/if}

			{#if errorMessage}
				<div class="mt-6 text-sm text-destructive">
					{errorMessage}
				</div>
			{/if}
		</div>

		<div class="sticky bottom-4 mt-8">
			<div class="rounded-2xl border bg-[#101010] px-5 py-4">
				<textarea
						bind:value={prompt}
						onkeydown={handleKeyDown}
						rows="3"
						placeholder="Какие способы закачки шахтных вод в глубокие горизонты применялись в России и за рубежом, и каковы их технико-экономические показатели?"
						class="field-sizing-content min-h-32 w-full resize-none border-0 bg-transparent p-0 text-[17px] leading-7 text-foreground placeholder:text-muted-foreground outline-none focus-visible:ring-0"
				></textarea>

				<div class="mt-3 flex items-center justify-between">
					<button
							type="button"
							class={useDomesticSources
							? "inline-flex items-center gap-2 rounded-full bg-foreground/10 px-3 py-2 text-sm text-foreground transition-colors"
							: "inline-flex items-center gap-2 rounded-full bg-muted/30 px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted/50"}
							onclick={() => (useDomesticSources = !useDomesticSources)}
					>
						<GlobeIcon class="size-4" />
						{useDomesticSources ? "Отечественные источники" : "Все источники"}
					</button>

					<Button
							type="button"
							size="icon"
							class="h-10 w-10 rounded-full"
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
	</section>
</main>
