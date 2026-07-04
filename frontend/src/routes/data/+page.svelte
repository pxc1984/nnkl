<script lang="ts">
	import { browser } from "$app/environment";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page } from "$app/state";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { listDataTags, listKnowledgeObjects } from "$lib/api/data";
	import DataEmptyState from "$lib/components/data/data-empty-state.svelte";
	import DataListItem from "$lib/components/data/data-list-item.svelte";
	import DataPageHeader from "$lib/components/data/data-page-header.svelte";
	import { Button } from "$lib/components/ui/button/index.js";
	import * as Card from "$lib/components/ui/card/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import { Badge } from "$lib/components/ui/badge/index.js";
	import type { DataTag, KnowledgeObject, PaginationMeta } from "$lib/data/types";
	import {
		buildDataSearchParams,
		DEFAULT_DATA_PAGE_SIZE,
		getTagsFromSearchParams,
		parseTagsInput,
	} from "$lib/data/utils";
	import { SearchIcon, XIcon } from "@lucide/svelte";

	let objects = $state<KnowledgeObject[]>([]);
	let meta = $state<PaginationMeta>({ page: 1, pageSize: DEFAULT_DATA_PAGE_SIZE, total: 0, totalPages: 0 });
	let availableTags = $state<DataTag[]>([]);
	let queryInput = $state("");
	let typeInput = $state("");
	let tagsInput = $state("");
	let isLoading = $state(false);
	let isLoadingTags = $state(false);
	let errorMessage = $state("");
	let requestRun = 0;
	let tagsLoaded = false;

	function syncFilterInputs(searchParams: URLSearchParams): void {
		queryInput = searchParams.get("query") ?? "";
		typeInput = searchParams.get("type") ?? "";
		tagsInput = getTagsFromSearchParams(searchParams).join(", ");
	}

	async function loadObjects(searchParams: URLSearchParams): Promise<void> {
		const currentRun = ++requestRun;
		isLoading = true;
		errorMessage = "";

		try {
			const response = await listKnowledgeObjects({
				page: Number(searchParams.get("page") || 1),
				pageSize: Number(searchParams.get("pageSize") || DEFAULT_DATA_PAGE_SIZE),
				query: searchParams.get("query") ?? undefined,
				type: searchParams.get("type") ?? undefined,
				tags: getTagsFromSearchParams(searchParams),
			});

			if (currentRun !== requestRun) {
				return;
			}

			objects = response.items;
			meta = response.meta;
		} catch (error) {
			if (currentRun !== requestRun) {
				return;
			}

			errorMessage = getApiErrorMessage(error, "Не удалось загрузить документы.");
			objects = [];
			meta = { page: 1, pageSize: DEFAULT_DATA_PAGE_SIZE, total: 0, totalPages: 0 };
		} finally {
			if (currentRun === requestRun) {
				isLoading = false;
			}
		}
	}

	async function loadTags(): Promise<void> {
		isLoadingTags = true;

		try {
			const response = await listDataTags();
			availableTags = response.items.slice(0, 8);
		} catch {
			availableTags = [];
		} finally {
			isLoadingTags = false;
		}
	}

	$effect(() => {
		syncFilterInputs(page.url.searchParams);

		if (!browser) {
			return;
		}

		void loadObjects(page.url.searchParams);
	});

	$effect(() => {
		if (!browser || tagsLoaded) {
			return;
		}

		tagsLoaded = true;
		void loadTags();
	});

	async function updateRoute(overrides: Partial<{
		page: number;
		pageSize: number;
		query: string;
		type: string;
		tags: string[];
	}>): Promise<void> {
		const searchParams = buildDataSearchParams({
			page: overrides.page ?? Number(page.url.searchParams.get("page") || 1),
			pageSize: overrides.pageSize ?? Number(page.url.searchParams.get("pageSize") || DEFAULT_DATA_PAGE_SIZE),
			query: overrides.query ?? queryInput,
			type: overrides.type ?? typeInput,
			tags: overrides.tags ?? parseTagsInput(tagsInput),
		});

		const search = searchParams.toString();
		await goto(search ? resolve(`/data?${search}` as `/data?${string}`) : resolve("/data"), {
			keepFocus: true,
			noScroll: true,
		});
	}

	async function handleFilterSubmit(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		await updateRoute({ page: 1 });
	}

	async function clearFilters(): Promise<void> {
		queryInput = "";
		typeInput = "";
		tagsInput = "";
		await updateRoute({ page: 1, query: "", type: "", tags: [] });
	}

	async function toggleQuickTag(tag: string): Promise<void> {
		const currentTags = parseTagsInput(tagsInput);
		const nextTags = currentTags.includes(tag)
			? currentTags.filter((currentTag) => currentTag !== tag)
			: [...currentTags, tag];

		tagsInput = nextTags.join(", ");
		await updateRoute({ page: 1, tags: nextTags });
	}

	const selectedTags = $derived(parseTagsInput(tagsInput));
	const totalPages = $derived(meta.totalPages || Math.max(1, Math.ceil(meta.total / Math.max(meta.pageSize, 1))));
</script>

<div class="flex flex-col gap-8">
	<DataPageHeader
		title="Документы"
		description="Загруженные и доступные вам материалы. Используйте поиск и фильтры, чтобы быстро находить нужные документы."
	>
		{#snippet actions()}
			<Button href="/data/upload" class="rounded-full">Загрузить</Button>
		{/snippet}
	</DataPageHeader>

	<Card.Root class="bg-card/90 rounded-[1.75rem] border-border/60 shadow-[0_20px_60px_-36px_rgba(0,0,0,0.35)]">
		<Card.Header class="gap-4 border-b border-border/60 pb-5">
			<div class="space-y-1">
				<Card.Title class="text-lg">Поиск и фильтры</Card.Title>
				<Card.Description>Состояние фильтров сохраняется в URL и им можно поделиться.</Card.Description>
			</div>
		</Card.Header>
		<Card.Content class="space-y-5 pt-5">
			<form class="grid gap-4 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,0.8fr)_minmax(0,1fr)_auto]" onsubmit={handleFilterSubmit}>
				<div class="relative">
					<SearchIcon class="text-muted-foreground pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2" />
					<Input bind:value={queryInput} placeholder="Название, автор или часть документа" class="h-10 rounded-full pl-9" />
				</div>
				<Input bind:value={typeInput} placeholder="Тип файла: pdf, docx" class="h-10 rounded-full" />
				<Input bind:value={tagsInput} placeholder="Теги через запятую" class="h-10 rounded-full" />
				<div class="flex gap-2">
					<Button type="submit" class="h-10 rounded-full">Применить</Button>
					<Button type="button" variant="ghost" class="h-10 rounded-full" onclick={clearFilters}>Очистить</Button>
				</div>
			</form>

			<div class="flex flex-wrap gap-2">
				{#if isLoadingTags}
					{#each [0, 1, 2, 3] as index (index)}
						<Skeleton class="h-8 w-24 rounded-full" />
					{/each}
				{:else}
					{#each availableTags as tag (tag.name)}
						<button
							type="button"
							class={selectedTags.includes(tag.name)
								? "bg-primary text-primary-foreground inline-flex h-8 items-center gap-2 rounded-full px-3 text-sm"
								: "bg-muted text-muted-foreground hover:bg-muted/80 inline-flex h-8 items-center gap-2 rounded-full px-3 text-sm transition-colors"}
							onclick={() => void toggleQuickTag(tag.name)}
						>
							<span>{tag.name}</span>
							<Badge variant={selectedTags.includes(tag.name) ? "secondary" : "outline"}>{tag.count}</Badge>
						</button>
					{/each}
				{/if}
			</div>
		</Card.Content>
	</Card.Root>

	<div class="flex flex-col gap-4">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<p class="text-foreground text-sm font-medium">{meta.total} документов</p>
				<p class="text-muted-foreground text-sm">
					Страница {meta.page} из {totalPages}
				</p>
			</div>

			{#if selectedTags.length > 0}
				<div class="flex flex-wrap gap-2">
					{#each selectedTags as tag (tag)}
						<button type="button" class="bg-muted inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm" onclick={() => void toggleQuickTag(tag)}>
							{tag}
							<XIcon class="size-3.5" />
						</button>
					{/each}
				</div>
			{/if}
		</div>

		{#if errorMessage}
			<div class="text-destructive bg-destructive/10 rounded-2xl border border-destructive/20 px-4 py-3 text-sm">{errorMessage}</div>
		{:else if isLoading}
			<div class="space-y-4">
				{#each [0, 1, 2, 3] as index (index)}
					<div class="bg-card/90 rounded-[1.5rem] border border-border/60 px-5 py-5">
						<Skeleton class="h-6 w-64 rounded-full" />
						<div class="mt-4 flex flex-wrap gap-3">
							<Skeleton class="h-4 w-28 rounded-full" />
							<Skeleton class="h-4 w-24 rounded-full" />
							<Skeleton class="h-4 w-40 rounded-full" />
						</div>
					</div>
				{/each}
			</div>
		{:else if objects.length === 0}
			<DataEmptyState
				title="Документы не найдены"
				description="Попробуйте изменить строку поиска или убрать часть фильтров. Если база еще пуста, начните с загрузки первого документа."
				actionLabel="Загрузить документы"
				actionHref="/data/upload"
			/>
		{:else}
			<div class="space-y-4">
				{#each objects as object (object.id)}
					<DataListItem {object} />
				{/each}
			</div>
		{/if}
	</div>

	{#if !isLoading && !errorMessage && meta.total > 0}
		<div class="flex flex-col gap-3 border-t border-border/60 pt-6 sm:flex-row sm:items-center sm:justify-between">
			<p class="text-muted-foreground text-sm">Показываем {objects.length} из {meta.total}</p>
			<div class="flex gap-2">
				<Button variant="outline" class="rounded-full" disabled={meta.page <= 1} onclick={() => void updateRoute({ page: meta.page - 1 })}>
					Назад
				</Button>
				<Button variant="outline" class="rounded-full" disabled={meta.page >= totalPages} onclick={() => void updateRoute({ page: meta.page + 1 })}>
					Далее
				</Button>
			</div>
		</div>
	{/if}
</div>