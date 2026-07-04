<script lang="ts">
	import { browser } from "$app/environment";
	import { getApiErrorMessage } from "$lib/api/auth";
	import { deleteKnowledgeObject, listKnowledgeObjects } from "$lib/api/data";
	import DataTableActions from "$lib/components/data/data-table-actions.svelte";
	import DataStatusBadge from "$lib/components/data/data-status-badge.svelte";
	import { Button } from "$lib/components/ui/button/index.js";
	import { Input } from "$lib/components/ui/input/index.js";
	import { Skeleton } from "$lib/components/ui/skeleton/index.js";
	import * as Table from "$lib/components/ui/table/index.js";
	import {
		FlexRender,
		createSvelteTable,
		renderComponent,
		renderSnippet,
	} from "$lib/components/ui/data-table/index.js";
	import type {
		ColumnDef,
		PaginationState,
	} from "@tanstack/table-core";
	import { getCoreRowModel } from "@tanstack/table-core";
	import type { KnowledgeObject, PaginationMeta } from "$lib/data/types";
	import {
		formatBytes,
		formatDateTime,
		getObjectTitle,
		parseTagsInput,
	} from "$lib/data/utils";
	import { cn } from "$lib/utils.js";
	import { createRawSnippet } from "svelte";
	import { ChevronLeftIcon, ChevronRightIcon } from "@lucide/svelte";

	const PAGE_SIZE = 20;
	const FILE_TYPE_OPTIONS = [
		{ value: "", label: "Все типы" },
		{ value: "pdf", label: "PDF" },
		{ value: "docx", label: "DOCX" },
		{ value: "pptx", label: "PPTX" },
		{ value: "markdown", label: "Markdown" },
	];

	let objects = $state<KnowledgeObject[]>([]);
	let paginationMeta = $state<PaginationMeta>({ page: 1, pageSize: PAGE_SIZE, total: 0, totalPages: 1 });
	let isLoading = $state(false);
	let errorMessage = $state("");
	let currentPage = $state(1);
	let pagination = $state<PaginationState>({ pageIndex: 0, pageSize: PAGE_SIZE });
	let confirmingDeleteId = $state<string | null>(null);
	let deletingId = $state<string | null>(null);
	let requestRun = 0;
	let deleteConfirmTimeout = $state<number | null>(null);
	let queryInput = $state("");
	let appliedQuery = $state("");
	let typeFilter = $state("");
	let tagsInput = $state("");
	let queryDebounceTimeout: number | null = null;

	const tagFilters = $derived(parseTagsInput(tagsInput));
	const hasActiveFilters = $derived(Boolean(appliedQuery || typeFilter || tagFilters.length > 0));

	function clearDeleteConfirmation(): void {
		if (deleteConfirmTimeout !== null) {
			window.clearTimeout(deleteConfirmTimeout);
			deleteConfirmTimeout = null;
		}

		confirmingDeleteId = null;
	}

	function armDeleteConfirmation(id: string): void {
		clearDeleteConfirmation();
		confirmingDeleteId = id;
		deleteConfirmTimeout = window.setTimeout(() => {
			if (confirmingDeleteId === id) {
				confirmingDeleteId = null;
				deleteConfirmTimeout = null;
			}
		}, 2000);
	}

	async function handleDeleteClick(id: string): Promise<void> {
		if (deletingId !== null) {
			return;
		}

		if (confirmingDeleteId !== id) {
			armDeleteConfirmation(id);
			return;
		}

		clearDeleteConfirmation();
		deletingId = id;
		errorMessage = "";

		const nextPage = objects.length === 1 && currentPage > 1 ? currentPage - 1 : currentPage;

		try {
			await deleteKnowledgeObject(id);

			if (nextPage !== currentPage) {
				currentPage = nextPage;
				pagination = { ...pagination, pageIndex: nextPage - 1 };
			} else {
				await loadData(nextPage);
			}
		} catch (error) {
			errorMessage = getApiErrorMessage(error, "Не удалось удалить материал.");
		} finally {
			deletingId = null;
		}
	}

	async function loadData(
		pageNum: number,
		filters: {
			query: string;
			type: string;
			tags: string[];
		} = {
			query: appliedQuery,
			type: typeFilter,
			tags: tagFilters,
		},
	): Promise<void> {
		const currentRun = ++requestRun;
		isLoading = true;
		errorMessage = "";

		try {
			const response = await listKnowledgeObjects({
				page: pageNum,
				pageSize: PAGE_SIZE,
				query: filters.query || undefined,
				type: filters.type || undefined,
				tags: filters.tags.length > 0 ? filters.tags : undefined,
			});
			if (currentRun !== requestRun) {
				return;
			}

			objects = response.items;
			paginationMeta = response.meta;
		} catch (error) {
			if (currentRun !== requestRun) {
				return;
			}

			errorMessage = getApiErrorMessage(error, "Не удалось загрузить список материалов.");
			objects = [];
		} finally {
			if (currentRun === requestRun) {
				isLoading = false;
			}
		}
	}

	function resetToFirstPage(): void {
		if (currentPage === 1 && pagination.pageIndex === 0) {
			return;
		}

		currentPage = 1;
		pagination = { ...pagination, pageIndex: 0 };
	}

	function handleTypeFilterChange(value: string): void {
		typeFilter = value;
		resetToFirstPage();
	}

	function handleTagsInput(value: string): void {
		tagsInput = value;
		resetToFirstPage();
	}

	function clearFilters(): void {
		queryInput = "";
		appliedQuery = "";
		typeFilter = "";
		tagsInput = "";
		resetToFirstPage();
	}

	$effect(() => {
		if (!browser) {
			return;
		}

		if (queryDebounceTimeout !== null) {
			window.clearTimeout(queryDebounceTimeout);
		}

		queryDebounceTimeout = window.setTimeout(() => {
			appliedQuery = queryInput.trim();
			queryDebounceTimeout = null;
		}, 300);

		return () => {
			if (queryDebounceTimeout !== null) {
				window.clearTimeout(queryDebounceTimeout);
			}
		};
	});

	$effect(() => {
		if (!browser) {
			return;
		}

		const pageNum = currentPage;
		const query = appliedQuery;
		const type = typeFilter;
		const tags = [...tagFilters];

		void loadData(pageNum, { query, type, tags });
	});

	$effect(() => {
		return () => {
			if (deleteConfirmTimeout !== null) {
				window.clearTimeout(deleteConfirmTimeout);
			}

			if (queryDebounceTimeout !== null) {
				window.clearTimeout(queryDebounceTimeout);
			}
		};
	});

	const totalPages = $derived(paginationMeta.totalPages ?? 1);
	const totalItems = $derived(paginationMeta.total ?? 0);

	const titleSnippet = createRawSnippet<[{ title: string }]>((getTitle) => {
		const { title } = getTitle();
		return {
			render: () => `<span class="font-medium block truncate">${title}</span>`,
		};
	});

	const sizeSnippet = createRawSnippet<[{ size: string }]>((getSize) => {
		const { size } = getSize();
		return {
			render: () => `<span class="text-muted-foreground">${size}</span>`,
		};
	});

	const dateSnippet = createRawSnippet<[{ date: string }]>((getDate) => {
		const { date } = getDate();
		return {
			render: () => `<span class="text-muted-foreground">${date}</span>`,
		};
	});

	const columns: ColumnDef<KnowledgeObject>[] = [
		{
			accessorKey: "filename",
			header: "Название",
			cell: ({ row }) => renderSnippet(titleSnippet, { title: getObjectTitle(row.original) }),
		},
		{
			accessorKey: "size",
			header: "Размер",
			cell: ({ row }) => renderSnippet(sizeSnippet, { size: formatBytes(row.original.size) }),
		},
		{
			accessorKey: "status",
			header: "Статус",
			cell: ({ row }) => renderComponent(DataStatusBadge, { status: row.original.status }),
		},
		{
			accessorKey: "createdAt",
			header: "Загружен",
			cell: ({ row }) => renderSnippet(dateSnippet, { date: formatDateTime(row.original.createdAt) }),
		},
		{
			id: "actions",
			header: "",
			cell: ({ row }) =>
				renderComponent(DataTableActions, {
					id: row.original.id,
					isDeleteConfirming: confirmingDeleteId === row.original.id,
					isDeleting: deletingId === row.original.id,
					onDeleteClick: (id: string) => void handleDeleteClick(id),
				}),
		},
	];

	const table = createSvelteTable({
		get data() {
			return objects;
		},
		columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: true,
		get pageCount() {
			return totalPages;
		},
		state: {
			get pagination() {
				return pagination;
			},
		},
		onPaginationChange: (updater) => {
			if (typeof updater === "function") {
				pagination = updater(pagination);
			} else {
				pagination = updater;
			}
			currentPage = pagination.pageIndex + 1;
		},
	});
</script>

<div class="flex flex-col gap-8 py-6">
	{#if errorMessage}
		<div class="text-sm text-destructive">
			{errorMessage}
		</div>
	{/if}

	<div class="flex flex-col gap-3 rounded-2xl border border-border/50 p-4 md:flex-row md:items-end">
		<div class="flex-1 space-y-1.5">
			<label class="text-sm font-medium" for="data-query-filter">Поиск</label>
			<Input
				id="data-query-filter"
				placeholder="Название файла..."
				bind:value={queryInput}
				oninput={resetToFirstPage}
				class="max-w-none"
			/>
		</div>

		<div class="space-y-1.5 md:w-48">
			<label class="text-sm font-medium" for="data-type-filter">Тип</label>
			<select
				id="data-type-filter"
				class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-8 w-full rounded-md border px-3 py-1 text-sm focus-visible:ring-2 focus-visible:outline-none"
				value={typeFilter}
				onchange={(event) => handleTypeFilterChange(event.currentTarget.value)}
			>
				{#each FILE_TYPE_OPTIONS as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>
		</div>

		<div class="flex-1 space-y-1.5">
			<label class="text-sm font-medium" for="data-tags-filter">Теги</label>
			<Input
				id="data-tags-filter"
				placeholder="tag1, tag2"
				value={tagsInput}
				oninput={(event) => handleTagsInput(event.currentTarget.value)}
				onchange={(event) => handleTagsInput(event.currentTarget.value)}
				class="max-w-none"
			/>
		</div>

		<Button
			variant="outline"
			disabled={!hasActiveFilters}
			onclick={clearFilters}
			class="md:self-end"
		>
			Сбросить
		</Button>
	</div>

	{#if isLoading}
		<div>
			<div class="flex items-center gap-4 border-b border-border/10 pb-3">
				<Skeleton class="h-5 w-48" />
				<Skeleton class="h-5 w-20" />
				<Skeleton class="h-5 w-16" />
				<Skeleton class="h-5 w-20" />
				<Skeleton class="ms-auto h-8 w-24" />
			</div>
			{#each [1, 2, 3, 4, 5] as i (i)}
				<div class="flex items-center gap-4 border-b border-border/5 py-3">
					<Skeleton class="h-5 w-48" />
					<Skeleton class="h-5 w-20" />
					<Skeleton class="h-5 w-16" />
					<Skeleton class="h-5 w-20" />
					<Skeleton class="ms-auto h-8 w-24" />
				</div>
			{/each}
		</div>
	{:else}
		<Table.Root class="table-fixed">
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row>
						{#each headerGroup.headers as header (header.id)}
							<Table.Head class={cn("has-[[role=checkbox]]:ps-3", header.column.id === "mimeType" && "hidden md:table-cell w-24", header.column.id === "filename" && "w-full min-w-0", header.column.id === "size" && "w-20", header.column.id === "status" && "w-24", header.column.id === "createdAt" && "w-36", header.column.id === "actions" && "w-36")}>
								{#if !header.isPlaceholder}
									<FlexRender
										content={header.column.columnDef.header}
										context={header.getContext()}
									/>
								{/if}
							</Table.Head>
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#each table.getRowModel().rows as row (row.id)}
					<Table.Row>
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell class={cn("has-[[role=checkbox]]:ps-3", cell.column.id === "mimeType" && "hidden md:table-cell", cell.column.id === "filename" && "min-w-0")}>
								<FlexRender
									content={cell.column.columnDef.cell}
									context={cell.getContext()}
								/>
							</Table.Cell>
						{/each}
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={columns.length} class="h-24 text-center">
							{hasActiveFilters ? "Нет результатов по выбранным фильтрам." : "Нет результатов."}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>

		<div class="flex items-center justify-between pt-2">
			<p class="text-muted-foreground text-sm">
				{totalItems === 1
					? "1 материал"
					: `${totalItems} материалов`}
			</p>

			<div class="flex items-center gap-1">
				<Button
					variant="outline"
					size="sm"
					disabled={!table.getCanPreviousPage()}
					onclick={() => table.previousPage()}
				>
					<ChevronLeftIcon class="size-4" />
				</Button>

				{#each Array.from({ length: Math.min(totalPages, 7) }, (__, i) => {
					const start = Math.max(0, Math.min(currentPage - 4, totalPages - 7));
					return start + i + 1;
				}) as pageNum (pageNum)}
					<Button
						variant={pageNum === currentPage ? "default" : "ghost"}
						size="sm"
						class="min-w-9"
						onclick={() => table.setPageIndex(pageNum - 1)}
					>
						{pageNum}
					</Button>
				{/each}

				<Button
					variant="outline"
					size="sm"
					disabled={!table.getCanNextPage()}
					onclick={() => table.nextPage()}
				>
					<ChevronRightIcon class="size-4" />
				</Button>
			</div>
		</div>
	{/if}
</div>
