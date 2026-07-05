<script lang="ts">
	import {browser} from "$app/environment";
	import {prependQuerySession} from "$lib/ask/query-sessions";
	import PrismaticBurst from "$lib/components/PrismaticBurst.svelte";
	import {Button} from "$lib/components/ui/button/index.js";
	import MarkdownRenderer from "$lib/components/markdown-renderer.svelte";
	import {ArrowUpIcon, GlobeIcon, FileTextIcon} from "@lucide/svelte";
	import {askQuestion, type AskResponse, type Reference} from "$lib/api/ask";
	import {getApiErrorMessage} from "$lib/api/auth";
	import {goto} from "$app/navigation";
	import {resolve} from "$app/paths";
	import {SvelteSet} from "svelte/reactivity";
	import {onMount} from "svelte";

	interface WebGLDebugRendererInfo {
		UNMASKED_RENDERER_WEBGL: number;
	}

	interface NavigatorWithDeviceMemory extends Navigator {
		deviceMemory?: number;
	}

	let prompt = $state("");
	let useDomesticSources = $state(false);
	let isLoading = $state(false);
	let answer = $state<AskResponse | null>(null);
	let errorMessage = $state("");
	let hasSubmittedPrompt = $state(false);
	let showBurst = $state(false);

	onMount(() => {
		showBurst = hasGoodDiscreteGpu();
	});

	function hasGoodDiscreteGpu(): boolean {
		if (!browser) {
			return false;
		}

		const canvas = document.createElement("canvas");
		const gl = canvas.getContext("webgl") ?? canvas.getContext("experimental-webgl");
		if (!gl) {
			return false;
		}

		const hasWebgl2 = !!canvas.getContext("webgl2");

		const debugInfo = gl.getExtension("WEBGL_debug_renderer_info") as WebGLDebugRendererInfo | null;
		const renderer = debugInfo
			? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)
			: gl.getParameter(gl.RENDERER);

		if (typeof renderer !== "string") {
			return false;
		}

		const normalizedRenderer = renderer.toLowerCase();
		const isSoftwareRenderer = /(llvmpipe|softpipe|swiftshader|software|virgl|basic render)/.test(normalizedRenderer);
		const looksIntegrated = /(intel|iris|uhd|apple|mali|adreno|powervr)/.test(normalizedRenderer);
		const looksDiscrete = /(nvidia|geforce|quadro|rtx|gtx|tesla|amd|radeon|firepro|w[0-9]{3,4})/.test(normalizedRenderer);
		const looksLowEndDiscrete = /(mx\d+|gt\s?\d+|rx\s?5\d{2}|r7\s|r5\s|pro\s?4\d{2}|t\d{3,4})/.test(normalizedRenderer);
		const browserNavigator = navigator as NavigatorWithDeviceMemory;
		const hasEnoughCpu = navigator.hardwareConcurrency >= 8;
		const hasEnoughMemory = (browserNavigator.deviceMemory ?? 0) >= 8;

		return !isSoftwareRenderer && !looksIntegrated && looksDiscrete && !looksLowEndDiscrete && hasWebgl2 && (hasEnoughCpu || hasEnoughMemory);
	}

	async function handleSubmit() {
		const query = prompt.trim();
		if (!query || isLoading) {
			return;
		}

		isLoading = true;
		hasSubmittedPrompt = true;
		errorMessage = "";
		answer = null;

		try {
			const mode = "naive";
			answer = await askQuestion(query, mode, useDomesticSources ? "ru" : undefined);

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
			let language = '';

			if (typeof ref.id === 'string' && isValidDocumentId(ref.id)) {
				// Enriched format returned by backend.
				id = ref.id.toLowerCase();
				filename = typeof ref.filename === 'string' ? ref.filename : '';
				type = typeof ref.type === 'string' ? ref.type : '';
				createdAt = typeof ref.createdAt === 'string' ? ref.createdAt : '';
				language = typeof ref.language === 'string' ? ref.language : '';
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
			result.push({ id, filename, type, createdAt, language, link: `/data/${id}` });
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

<main class="relative flex flex-1 overflow-hidden bg-[#101010]">
	{#if !hasSubmittedPrompt}
		<div class="pointer-events-none absolute inset-0">
			{#if showBurst}
				<div class="absolute inset-0 opacity-75">
					<PrismaticBurst
						intensity={1.75}
						speed={0.5}
						animationType="rotate3d"
						distort={0.35}
						hoverDampness={0.12}
						rayCount={8}
						colors={["#38bdf8", "#818cf8", "#c084fc", "#f472b6"]}
						mixBlendMode="screen"
					/>
				</div>
			{/if}
			<div class="absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(8,12,18,0)_0%,_rgba(8,12,18,0.5)_42%,_rgba(8,12,18,0.88)_100%)]"></div>
		</div>
	{/if}

	<section class="relative z-10 mx-auto flex w-full max-w-3xl flex-1 flex-col px-4 pb-6 pt-20 md:px-8">
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
						{#if useDomesticSources}
							<span class="flag-tricolor size-4 rounded-sm">
								<span class="flag-stripe flag-stripe--white"></span>
								<span class="flag-stripe flag-stripe--blue"></span>
								<span class="flag-stripe flag-stripe--red"></span>
							</span>
							Отечественные источники
						{:else}
							<GlobeIcon class="size-4" />
							Все источники
						{/if}
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

<style>
	.flag-tricolor {
		display: flex;
		flex-direction: column;
		overflow: hidden;
		flex-shrink: 0;
		animation: flag-wave 1.8s ease-in-out infinite;
	}

	.flag-stripe {
		flex: 1;
		width: 100%;
	}

	.flag-stripe--white {
		background: #fff;
	}

	.flag-stripe--blue {
		background: #0039a6;
	}

	.flag-stripe--red {
		background: #d52b1e;
	}

	@keyframes flag-wave {
		0%, 100% {
			transform: skewX(0deg) scaleY(1);
		}
		25% {
			transform: skewX(-2deg) scaleY(1.06);
		}
		75% {
			transform: skewX(2deg) scaleY(1.06);
		}
	}
</style>
